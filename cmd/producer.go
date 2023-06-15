/*****************************************************************************
*
*	Project			: TFM 2.0 -> FeatureSpace
*
*	File			: producer.go
*
* 	Created			: 27 Aug 2021
*
*	Description		:
*
*	Modified		: 27 Aug 2021	- Start
*					: 24 Jan 2023   - Mod for applab sandbox
*					: 20 Feb 2023	- repckaged for TFM 2.0 load generation, we're creating fake FS engineResponse messages onto a
*					:				- Confluent Kafka topic
*					: 1 June 2023	- ...
*
*	By				: George Leonard (georgelza@gmail.com)
*
*	jsonformatter 	: https://jsonformatter.curiousconcept.com/#
*
*****************************************************************************/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit"
	"github.com/google/uuid"
	"github.com/tkanos/gonfig"
	"golang.org/x/exp/slog"

	"github.com/TylerBrock/colorjson"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	// Prometheus
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// My Types/Structs/functions
	"cmd/types"
)

// sql fetch from backend data store
var sql_duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "fs_sql_duration_seconds",
	Help:    "Histogram for the duration in seconds.",
	Buckets: []float64{1, 2, 5, 6, 10},
},
	[]string{"endpoint"},
)

// entire json body build and api post
var rec_duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "fs_record_duration_seconds",
	Help:    "Histogram for the duration in seconds for entire record.",
	Buckets: []float64{1, 2, 5, 6, 10},
},
	[]string{"endpoint"},
)

// api call duration, aka time to post record onto fs api endpoint
var api_duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "fs_api_duration_seconds",
	Help:    "Histogram for the duration in seconds for api call.",
	Buckets: []float64{1, 2, 5, 6, 10},
},
	[]string{"endpoint"},
)

// current number of executes/api posts
var req_processed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "fs_etl_operations_count",
	Help: "The total number of processed records",
},
	[]string{"batch"},
)

// total number of records returned by sql fetch that need to be processed
var info = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "txn_count",
	Help: "Information about the batch size",
},
	[]string{"batch"},
)

var (
	logger   *slog.Logger
	vGeneral types.TGeneral
	vKafka   types.TKafka
	varSeed  types.TPSeed
)

func init() {

	// Keeping it very simple
	//grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)

	mode := "text"

	var programLevel = new(slog.LevelVar) // Info by default
	if mode == "text" {
		mHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
		logger = slog.New(mHandler)

	} else { // JSON
		mHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
		logger = slog.New(mHandler)

	}
	slog.SetDefault(logger)
	programLevel.Set(slog.LevelDebug)

	log.Println("###############################################################")
	log.Println("#")
	log.Println("#   Project   : TFM 2.0")
	log.Println("#")
	log.Println("#   Comment   : FeatureSpace Scenario Publisher / Fake Data Generator")
	log.Println("#             : This application posts completed events and engineResponses onto")
	log.Println("#             : Confluent hosted Kafka topics")
	log.Println("#")
	log.Println("#   By        : George Leonard (georgelza@gmail.com)")
	log.Println("#")
	log.Println("#   Date/Time :", time.Now().Format("27-01-2023 - 15:04:05"))
	log.Println("#")
	log.Println("###############################################################")
	log.Println("")
	log.Println("")

	// Initialize the vGeneral struct variable.
	vGeneral = loadConfig("dev")
	vKafka = loadKafka("dev")
	varSeed = loadSeed(vGeneral.SeedFile)

	if vGeneral.EchoGeneralConfig == 1 {
		printConfig(vGeneral)

		v, err := json.Marshal(vKafka)
		if err != nil {
			logger.Error(fmt.Sprintf("Marchalling error: %s", err))
			os.Exit(1)
		}

		if vKafka.EchoKafkaConfig == 1 {
			prettyJSON(string(v))
		}

		v, err = json.Marshal(varSeed)
		if err != nil {
			logger.Error(fmt.Sprintf("Marchalling error: %s", err))
			os.Exit(1)
		}

		if vGeneral.EchoSeed == 1 {
			prettyJSON(string(v))

		}
	}
}

// Load General configuration Parameters
func loadConfig(params ...string) types.TGeneral {

	vGeneral := types.TGeneral{}
	env := "dev"
	if len(params) > 0 {
		env = params[0]
	}

	path, err := os.Getwd()
	if err != nil {
		logger.Error(fmt.Sprintf("Problem retrieving current path: %s", err))
		os.Exit(1)
	}

	//	fileName := fmt.Sprintf("%s/%s_app.json", path, env)
	fileName := fmt.Sprintf("%s/%s_app.json", path, env)

	err = gonfig.GetConf(fileName, &vGeneral)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Reading Config File: %s", err))
		os.Exit(1)

	} else {

		vHostname, err := os.Hostname()
		if err != nil {
			logger.Error(fmt.Sprintf("Can't retrieve hostname %s", err))
			os.Exit(1)

		}
		vGeneral.Hostname = vHostname

		if vGeneral.Json_to_file == 1 {
			vGeneral.Output_path = path + "/" + vGeneral.Output_path

		} else {
			vGeneral.Output_path = ""

		}

		if vGeneral.Json_from_file == 1 {
			vGeneral.Input_path = path + "/" + vGeneral.Input_path

		} else {
			vGeneral.Input_path = ""

		}
	}

	if vGeneral.Debuglevel > 1 {
		logger.Info("*")
		logger.Info("* General Config :")
		logger.Info(fmt.Sprintf("* Current path   : %s", path))
		logger.Info(fmt.Sprintf("* Config File    : %s", fileName))
		logger.Info("*")

	}

	return vGeneral
}

// Load Kafka specific configuration Parameters, this is so that we can gitignore this dev_kafka.json file/seperate
// from the dev_app.json file
func loadKafka(params ...string) types.TKafka {

	vKafka := types.TKafka{}
	env := "dev"
	if len(params) > 0 {
		env = params[0]
	}

	path, err := os.Getwd()
	if err != nil {
		logger.Error(fmt.Sprintf("Problem retrieving current path: %s", err))
		os.Exit(1)

	}

	//	fileName := fmt.Sprintf("%s/%s_app.json", path, env)
	fileName := fmt.Sprintf("%s/%s_kafka.json", path, env)
	err = gonfig.GetConf(fileName, &vKafka)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Reading Kafka File: %s", err))
		os.Exit(1)

	}

	if vGeneral.Debuglevel > 1 {

		logger.Info("*")
		logger.Info("* Kafka Config :")
		logger.Info(fmt.Sprintf("* Current path : %s", path))
		logger.Info(fmt.Sprintf("* Kafka File   : %s", fileName))
		logger.Info("*")

	}

	return vKafka
}

// load the seed data from seed.json
func loadSeed(file string) types.TPSeed {

	var vSeed types.TPSeed

	path, err := os.Getwd()
	if err != nil {
		logger.Error(fmt.Sprintf("Problem retrieving current path: %s", err))
		os.Exit(1)

	}

	fileName := fmt.Sprintf("%s/%s", path, file)
	err = gonfig.GetConf(fileName, &vSeed)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Reading Seed File: %s", err))
		os.Exit(1)
	}

	if vGeneral.Debuglevel > 1 {
		logger.Info("*")
		logger.Info("* Seed Data    :")
		logger.Info(fmt.Sprintf("* Current path : %s", path))
		logger.Info(fmt.Sprintf("* Seed File    : %s", fileName))
		logger.Info("*")
	}

	return vSeed
}

// print some configurations
func printConfig(vGeneral types.TGeneral) {

	log.Println("****** General Parameters *****")
	log.Println("*")
	log.Println("* Hostname is              ", vGeneral.Hostname)
	log.Println("* Log Level is             ", vGeneral.Loglevel)
	log.Println("* Debug Level is           ", vGeneral.Debuglevel)
	log.Println("* Echo JSON is             ", vGeneral.Echojson)
	log.Println("*")
	log.Println("* Sleep Duration is        ", vGeneral.Sleep)
	log.Println("* Test Batch Size is       ", vGeneral.Testsize)
	log.Println("* Event Type is            ", vGeneral.Eventtype)
	log.Println("* Output JSON to file is   ", vGeneral.Json_to_file)
	log.Println("* Output path is           ", vGeneral.Output_path)
	log.Println("* Read JSON from file is   ", vGeneral.Json_from_file)
	log.Println("* Input path is            ", vGeneral.Input_path)
	log.Println("* MinTransactionValue is  R", vGeneral.MinTransactionValue)
	log.Println("* MaxTransactionValue is  R", vGeneral.MaxTransactionValue)
	log.Println("* SeedFile is              ", vGeneral.SeedFile)
	log.Println("* EchoSeed is              ", vGeneral.EchoSeed)
	log.Println("*")
	log.Println("*******************************")

	log.Println("")

}

// print some more configurations
func printKafkaConfig(vKafka types.TKafka) {

	logger.Info("****** Kafka Connection Parameters *****")
	logger.Info("*")
	logger.Info("* Kafka bootstrap Server is\t", vKafka.Bootstrapservers)
	logger.Info("* Kafka Topic is\t\t", vKafka.Topicname)
	logger.Info("* Kafka # Parts is\t\t", vKafka.Numpartitions)
	logger.Info("* Kafka Rep Factor is\t\t", vKafka.Replicationfactor)
	logger.Info("* Kafka Retension is\t\t", vKafka.Retension)
	logger.Info("* Kafka ParseDuration is\t", vKafka.Parseduration)
	logger.Info("*")
	logger.Info("* Kafka Flush Size is\t\t", vKafka.Flush_interval)
	logger.Info("*")
	logger.Info("*******************************")

	logger.Info("")

}

// Create Kafka topic if not exist, using admin client
func CreateTopic(props types.TKafka) {

	cm := kafka.ConfigMap{
		"bootstrap.servers":       props.Bootstrapservers,
		"broker.version.fallback": "0.10.0.0",
		"api.version.fallback.ms": 0,
	}

	if props.Sasl_mechanisms != "" {
		cm["sasl.mechanisms"] = props.Sasl_mechanisms
		cm["security.protocol"] = props.Security_protocol
		cm["sasl.username"] = props.Sasl_username
		cm["sasl.password"] = props.Sasl_password
		if vGeneral.Debuglevel > 0 {
			logger.Info("* Security Authentifaction configured in ConfigMap")

		}
	}
	if vGeneral.Debuglevel > 0 {
		logger.Info("* Basic Client ConfigMap compiled")
	}

	adminClient, err := kafka.NewAdminClient(&cm)
	if err != nil {
		logger.Error(fmt.Sprintf("Admin Client Creation Failed: %s", err))
		os.Exit(1)

	}
	if vGeneral.Debuglevel > 0 {
		logger.Info("* Admin Client Created Succeeded")

	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxDuration, err := time.ParseDuration(props.Parseduration)
	if err != nil {
		logger.Error(fmt.Sprintf("Error Configuring maxDuration via ParseDuration: %s", props.Parseduration))
		os.Exit(1)

	}
	if vGeneral.Debuglevel > 0 {
		logger.Info("* Configured maxDuration via ParseDuration")

	}

	results, err := adminClient.CreateTopics(ctx,
		[]kafka.TopicSpecification{{
			Topic:             props.Topicname,
			NumPartitions:     props.Numpartitions,
			ReplicationFactor: props.Replicationfactor}},
		kafka.SetAdminOperationTimeout(maxDuration))

	if err != nil {
		logger.Error(fmt.Sprintf("Problem during the topic creation: %v", err))
		os.Exit(1)
	}

	// Check for specific topic errors
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError &&
			result.Error.Code() != kafka.ErrTopicAlreadyExists {
			logger.Error(fmt.Sprintf("Topic Creation Failed for %s: %v", result.Topic, result.Error.String()))
			os.Exit(1)

		} else {
			if vGeneral.Debuglevel > 0 {
				logger.Info(fmt.Sprintf("* Topic Creation Succeeded for %s", result.Topic))

			}
		}
	}

	adminClient.Close()
	logger.Info("")

}

// Pretty Print JSON string
func prettyJSON(ms string) {

	var obj map[string]interface{}
	//json.Unmarshal([]byte(string(m.Value)), &obj)
	json.Unmarshal([]byte(ms), &obj)

	// Make a custom formatter with indent set
	f := colorjson.NewFormatter()
	f.Indent = 4

	// Marshall the Colorized JSON
	result, _ := f.Marshal(obj)
	fmt.Println(string(result))

}

// paymentNRT payload build
func contructPaymentNRTFromFake() (t_Payment map[string]interface{}, toID string, cMerchant string) {

	// We just using gofakeit to pad the json document size a bit.
	//
	// https://github.com/brianvoe/gofakeit
	// https://pkg.go.dev/github.com/brianvoe/gofakeit

	gofakeit.Seed(time.Now().UnixNano())
	gofakeit.Seed(0)

	//tenantCount := len(varSeed.Tenants) - 1
	tenantCount := 4 // tenants - From Banks

	directionCount := len(varSeed.Direction) - 1
	cDirection := varSeed.Direction[gofakeit.Number(0, directionCount)]

	cTenant := gofakeit.Number(0, tenantCount) // tenants - to Bank
	jTenant := varSeed.Tenants[cTenant]

	cTo := gofakeit.Number(0, tenantCount)
	jTo := varSeed.Tenants[cTo]

	nAmount := gofakeit.Price(vGeneral.MinTransactionValue, vGeneral.MaxTransactionValue)
	t_amount := &types.TAmount{
		BaseCurrency: "zar",
		BaseValue:    nAmount,
		Currency:     "zar",
		Value:        nAmount,
	}

	// Are we using good or bad data
	var jMerchant types.TEntity2
	if vGeneral.IntroBadEntity == 0 {
		entityCount := len(varSeed.GoodEntities) - 1
		cMerchant := gofakeit.Number(0, entityCount)
		jMerchant = varSeed.GoodEntities[cMerchant]

	} else {
		entityCount := len(varSeed.BadEntities) - 1
		cMerchant := gofakeit.Number(0, entityCount)
		jMerchant = varSeed.BadEntities[cMerchant]

	}

	// Are we using good or bad data
	var jAgent types.TAgent
	if vGeneral.IntroBadAgent == 0 {
		agentCount := len(varSeed.GoodAgent) - 1
		cAgent := gofakeit.Number(0, agentCount)
		jAgent = varSeed.GoodAgent[cAgent]

	} else {
		agentCount := len(varSeed.BadAgent) - 1
		cAgent := gofakeit.Number(0, agentCount)
		jAgent = varSeed.BadAgent[cAgent]

	}

	// We ust showing 2 ways to construct a JSON document to be Marshalled, this is the first using a map/interface,
	// followed by using a set of struct objects added together.
	t_Payment = map[string]interface{}{
		"accountAgentId":                 jAgent.Id,
		"accountAgentName":               jAgent.Name,
		"accountEntityId":                strconv.Itoa(rand.Intn(6)),
		"accountId":                      jMerchant.EntityId,
		"amount":                         t_amount,
		"chargeBearer":                   "SLEV",
		"counterpartyAgentId":            "",
		"counterpartyEntityId":           strconv.Itoa(gofakeit.Number(0, 9999)),
		"counterpartyId":                 strconv.Itoa(gofakeit.Number(10000, 19999)),
		"customerEntityId":               "customerEntityId_1",
		"creationTime":                   time.Now().Format("2006-01-02T15:04:05"),
		"destinationCountry":             "ZAF",
		"direction":                      cDirection,
		"eventId":                        uuid.New().String(),
		"eventTime":                      time.Now().Format("2006-01-02T15:04:05"),
		"eventType":                      "paymentNRT",
		"fromFIBranchId":                 "",
		"fromId":                         jTenant.TenantId,
		"localInstrument":                "42",
		"msgStatus":                      "Success",
		"msgType":                        "RCCT",
		"numberOfTransactions":           1,
		"paymentClearingSystemReference": 2,
		"paymentMethod":                  "TRF",
		"paymentReference":               "sdfsfd",
		"remittanceId":                   "sdfsdsd",
		"requestExecutionDate":           time.Now().Format("2006-01-02"),
		"schemaVersion":                  1,
		"settlementClearingSystemCode":   "RTC",
		"settlementDate":                 time.Now().Format("2006-01-02"),
		"settlementMethod":               "CLRG",
		"tenantId":                       jTenant.TenantId,
		"toFIBranchId":                   jTo.TenantId,
		"toId":                           jTo.TenantId,
		"totalAmount":                    t_amount,
		"transactionId":                  uuid.New().String(),
	}

	return t_Payment, toID, cMerchant
}

// paymentRT payload build - bit more complicated/larger than paymentNRT
func contructPaymentRTFromFake() (t_Payment map[string]interface{}, toID string, cMerchant string) {

	// We just using gofakeit to pad the json document size a bit.
	//
	// https://github.com/brianvoe/gofakeit
	// https://pkg.go.dev/github.com/brianvoe/gofakeit

	gofakeit.Seed(time.Now().UnixNano())
	gofakeit.Seed(0)

	nAmount := gofakeit.Price(vGeneral.MinTransactionValue, vGeneral.MaxTransactionValue)
	t_amount := &types.TAmount{
		BaseCurrency: "zar",
		BaseValue:    nAmount,
		Currency:     "zar",
		Value:        nAmount,
	}

	directionCount := len(varSeed.Direction) - 1
	cDirection := gofakeit.Number(0, directionCount)
	direction := varSeed.Direction[cDirection]

	//tenantCount := len(varSeed.Tenants) - 1
	tenantCount := 3

	chargeBearersCount := len(varSeed.ChargeBearers) - 1
	goodEntityCount := len(varSeed.GoodEntities) - 1
	//	badEntityCount := len(varSeed.BadEntities) - 1
	//	goodPayerCount := len(varSeed.GoodPayers) - 1
	//	badPayerCount := len(varSeed.BadPayers) - 1

	paymentFrequencyCount := len(varSeed.PaymentFrequency) - 1
	remittanceLocationMethodCount := len(varSeed.RemittanceLocationMethod) - 1
	settlementMethodCount := len(varSeed.SettlementMethod) - 1
	transactionTypesCount := len(varSeed.TransactionTypesRt) - 1
	verificationResultCount := len(varSeed.VerificationResult) - 1

	cId := gofakeit.Number(0, tenantCount)
	jToID := varSeed.Tenants[cId] // tenants - to Bank
	jTenant := varSeed.Tenants[gofakeit.Number(0, tenantCount)]

	nChargeBearers := gofakeit.Number(0, chargeBearersCount)
	//cCounterPartyAgent := gofakeit.Number(0, agentsCount) // Agents
	nCounterParty := gofakeit.Number(0, goodEntityCount) // Agents
	nPaymentFrequency := gofakeit.Number(0, paymentFrequencyCount)
	nRemittanceLocationMethod := gofakeit.Number(0, remittanceLocationMethodCount)
	nSettlementMethod := gofakeit.Number(0, settlementMethodCount)
	nTransactionTypesRt := gofakeit.Number(3, transactionTypesCount) // 3 onwards is RT types
	nVerificationResult := gofakeit.Number(0, verificationResultCount)

	chargeBearers := varSeed.ChargeBearers[nChargeBearers]
	paymentFrequency := varSeed.PaymentFrequency[nPaymentFrequency]
	xtransactionTypes := varSeed.TransactionTypesRt[nTransactionTypesRt]
	xverificationResult := varSeed.VerificationResult[nVerificationResult]

	// Are we using good or bad data
	var jAgent types.TAgent
	var jCounterPartyAgent types.TAgent
	var jInstructedAgent types.TAgent
	var jInstructingAgent types.TAgent
	var jIntermediaryAgent1Id types.TAgent
	var jIntermediaryAgent2Id types.TAgent
	var jIntermediaryAgent3Id types.TAgent
	var jPayer types.TPayer

	agentCount := len(varSeed.GoodAgent) - 1

	cAgent := gofakeit.Number(0, agentCount)
	jAgent = varSeed.GoodAgent[cAgent]

	cCounterPartyAgent := gofakeit.Number(0, agentCount)
	jCounterPartyAgent = varSeed.GoodAgent[cCounterPartyAgent]

	cInstructedAgent := gofakeit.Number(0, agentCount)
	jInstructedAgent = varSeed.GoodAgent[cInstructedAgent]

	cInstructingAgent := gofakeit.Number(0, agentCount)
	jInstructingAgent = varSeed.GoodAgent[cInstructingAgent]

	cIntermediaryAgent1Id := gofakeit.Number(0, agentCount)
	jIntermediaryAgent1Id = varSeed.GoodAgent[cIntermediaryAgent1Id]

	cIntermediaryAgent2Id := gofakeit.Number(0, agentCount)
	jIntermediaryAgent2Id = varSeed.GoodAgent[cIntermediaryAgent2Id]

	cIntermediaryAgent3Id := gofakeit.Number(0, agentCount)
	jIntermediaryAgent3Id = varSeed.GoodAgent[cIntermediaryAgent3Id]

	payerCount := len(varSeed.GoodPayers) - 1
	cPayer := gofakeit.Number(0, payerCount)
	jPayer = varSeed.GoodPayers[cPayer]

	// We ust showing 2 ways to construct a JSON document to be Marshalled, this is the first using a map/interface,
	// followed by using a set of struct objects added together.
	t_Payment = map[string]interface{}{
		"accountAgentId":   jAgent.Id,
		"accountAgentName": jAgent.Name,
		"accountAgentAddress": map[string]interface{}{
			"addressLine1":       "accountAgentAddress_addressLine1_1",
			"addressLine2":       "accountAgentAddress_addressLine2_1",
			"addressLine3":       "accountAgentAddress_addressLine3_1",
			"addressType":        "accountAgentAddress_addressType_1",
			"country":            "acc",
			"countrySubDivision": "accountAgentAddress_countrySubDivision_1",
			"postalCode":         "accountAgentAddress_postalCode_1",
			"townName":           "accountAgentAddress_townName_1",
		},
		"accountId":           uuid.New().String(),
		"accountBalanceAfter": t_amount,
		//	"accountEntityId":     "",
		"accountAddress": map[string]interface{}{
			"addressLine1": "accountAddress_addressLine1_1",
			"addressLine2": "accountAddress_addressLine2_1",
			"postalCode":   "accountAddress_postalCode_1",
			"townName":     "accountAddress_townName_1",
			"country":      "ZAF",
		},
		"accountName": map[string]interface{}{
			"fullName":   jPayer.Name.FullName,
			"namePrefix": jPayer.Name.NamePrefix,
			"surname":    jPayer.Name.Surname,
		},
		"amount":       t_amount,
		"cardEntityId": strconv.Itoa(gofakeit.CreditCardNumber()) + "CarSupplier",
		"cardId":       strconv.Itoa(gofakeit.CreditCardNumber()),
		"channel":      "",
		"chargeBearer": chargeBearers,
		"counterpartyAddress": map[string]interface{}{
			"addressLine1": "counterpartyAddress_addressLine1_1",
			"addressLine2": "counterpartyAddress_addressLine2_1",
			"addressLine3": "counterpartyAddress_addressLine3_1",
			"country":      "ZAF",
			"postalCode":   "e1234",
		},
		"counterpartyAgentId":      jCounterPartyAgent.Id,
		"counterpartyAgentName":    jCounterPartyAgent.Name,
		"counterpartyAgentAddress": jCounterPartyAgent.Address,
		"counterpartyId":           varSeed.GoodEntities[nCounterParty].Id,
		"counterpartyName":         map[string]interface{}{},
		"counterpartyEntityId":     varSeed.GoodEntities[nCounterParty].EntityId,
		"creationDate":             time.Now().Format("2006-01-02T15:04:05"),
		"customerEntityId":         jPayer.AccountNumber,
		"customerId":               jPayer.Id,
		"decorationId":             varSeed.Decoration,
		"destinationCountry":       "ZAF",
		"device": map[string]interface{}{
			"anonymizerInUseFlag": "device_anonymizerInUseFlag_1",
			"deviceFingerprint":   "device_deviceFingerprint_1",
			"deviceIMEI":          "device_deviceIMEI_1",
			"sessionLatitude":     5391.835331574868,
			"sessionLongitude":    6200.391276149532,
		},
		"deviceEntityId":                      "",
		"deviceId":                            "IMEACode",
		"direction":                           direction,
		"eventId":                             uuid.New().String(),
		"eventTime":                           time.Now().Format("2006-01-02T15:04:05"),
		"eventType":                           "paymentRT",
		"finalPaymentDate":                    time.Now().Format("2006-01-02"),
		"firstPaymentDate":                    time.Now().Format("2006-01-02"),
		"fromFIBranchId":                      "",
		"fromId":                              jTenant.TenantId,
		"instructedAgentId":                   jInstructedAgent.Id,
		"instructedAgentName":                 jInstructedAgent.Name,
		"instructedAgentAddress":              jInstructedAgent.Address,
		"instructingAgentId":                  jInstructingAgent.Id,
		"instructingAgentName":                jInstructingAgent.Name,
		"instructingAgentAddress":             jInstructingAgent.Address,
		"intermediaryAgent1Id":                jIntermediaryAgent1Id.Id,
		"intermediaryAgent1Name":              jIntermediaryAgent1Id.Name,
		"intermediaryAgent1AccountId":         jIntermediaryAgent1Id.AccountId,
		"intermediaryAgent1Address":           jIntermediaryAgent1Id.Address,
		"intermediaryAgent2Id":                jIntermediaryAgent2Id.Id,
		"intermediaryAgent2Name":              jIntermediaryAgent2Id.Name,
		"intermediaryAgent2AccountId":         jIntermediaryAgent2Id.AccountId,
		"intermediaryAgent2Address":           jIntermediaryAgent2Id.Address,
		"intermediaryAgent3Id":                jIntermediaryAgent3Id.Id,
		"intermediaryAgent3Name":              jIntermediaryAgent3Id.Name,
		"intermediaryAgent3AccountId":         jIntermediaryAgent3Id.AccountId,
		"intermediaryAgent3Address":           jIntermediaryAgent3Id.Address,
		"localInstrument":                     "",
		"msgStatus":                           "New",
		"msgStatusReason":                     "New Payee",
		"msgType":                             "2100",
		"numberOfTransactions":                1,
		"paymentClearingSystemReference":      "",
		"paymentFrequency":                    paymentFrequency,
		"paymentMethod":                       "",
		"paymentReference":                    "",
		"remittanceId":                        "",
		"remittanceLocationElectronicAddress": "remittanceLocationElectronicAddress_1",
		"remittanceLocationMethod":            varSeed.RemittanceLocationMethod[nRemittanceLocationMethod],
		"requestExecutionDate":                time.Now().Format("2006-01-02"),
		"schemaVersion":                       1,
		"serviceLevelCode":                    "",
		"settlementClearingSystemCode":        "",
		"settlementDate":                      time.Now().Format("2006-01-02"),
		"settlementMethod":                    varSeed.SettlementMethod[nSettlementMethod],
		"tenantId":                            jTenant.TenantId,
		"toFIBranchId":                        jToID.TenantId,
		"toId":                                jToID.TenantId,
		"totalAmount":                         t_amount,
		"transactionId":                       uuid.New().String(),
		"transactionType":                     xtransactionTypes,
		"ultimateAccountAddress": map[string]interface{}{
			"addressLine1": "ultimateCounterpartyAddress_addressLine1_1",
			"addressLine2": "ultimateCounterpartyAddress_addressLine2_1",
			"addressLine3": "ultimateCounterpartyAddress_addressLine3_1",
			"postalCode":   "2342342",
			"country":      "ZAF",
		},
		"ultimateAccountId": "",
		"ultimateAccountName": map[string]interface{}{
			"fullName":  "ultimateAccountName_fullName_1",
			"givenName": "ultimateAccountName_givenName_1",
			"surname":   "ultimateAccountName_surname_1",
		},
		"ultimateCounterpartyAddress": map[string]interface{}{
			"addressLine1":       "ultimateCounterpartyAddress_addressLine1_1",
			"addressLine2":       "ultimateCounterpartyAddress_addressLine2_1",
			"countrySubDivision": "ultimateCounterpartyAddress_countrySubDivision_1",
			"latitude":           8534.86390567603,
			"longitude":          6842.924811568308,
			"postalCode":         "ultimateCounterpartyAddress_postalCode_1",
			"townName":           "ultimateCounterpartyAddress_townName_1",
			"country":            "ZAF",
		},
		"ultimateCounterpartyId": "",
		"ultimateCounterpartyName": map[string]interface{}{
			"fullName":  "ultimateCounterpartyName_fullName_1",
			"givenName": "ultimateCounterpartyName_givenName_1",
			"surname":   "ultimateCounterpartyName_surname_1",
		},
		"unstructuredRemittanceInformation": "",
		"verificationResult":                xverificationResult,
		"verificationType": map[string]interface{}{
			"aa": "verificationType_aa_1",
		},
	}

	return t_Payment, jToID.TenantId, cMerchant
}

// Fake Query database and get the record set to work with
func fetchEFTRecords() (records []string, count int) {

	if vGeneral.Debuglevel > 0 {
		logger.Info("**** Quering Backend database ****")

	}

	////////////////////////////////
	// start a timer
	sTime := time.Now()

	// Execute a large sql #1 execute
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10000) // if vGeneral.sleep = 10000, 10 second
	if vGeneral.Debuglevel > 0 {
		logger.Info(fmt.Sprintf("* SQL Execute, Sleeping %v Millisecond - Simulating long database fetch...", n))

	}
	time.Sleep(time.Duration(n) * time.Millisecond)

	// post to Prometheus
	sql_duration.WithLabelValues("eft").Observe(time.Since(sTime).Seconds())

	// Return pointer to recordset and counter of number of records
	count = 4252345123
	if vGeneral.Debuglevel > 0 {
		logger.Info("**** Backend dataset retrieved ****")

	}

	return records, count
}

func runLoader() {

	if vGeneral.Debuglevel > 0 {
		// Create admin client to create the topic if it does not exist
		logger.Info("**** Confirm Topic Existence & Configuration ****")
		logger.Info("*")

	}

	// Lets make sure the topic/s exist
	CreateTopic(vKafka)

	// --
	// Create Producer instance
	// https://docs.confluent.io/current/clients/confluent-kafka-go/index.html#NewProducer

	if vGeneral.Debuglevel > 0 {
		logger.Info("**** Configure Client Kafka Connection ****")
		logger.Info("*")
		logger.Info(fmt.Sprintf("* Kafka bootstrap Server is %s", vKafka.Bootstrapservers))

	}

	cm := kafka.ConfigMap{
		"bootstrap.servers":       vKafka.Bootstrapservers,
		"broker.version.fallback": "0.10.0.0",
		"api.version.fallback.ms": 0,
		"client.id":               vGeneral.Hostname,
	}

	if vGeneral.Debuglevel > 0 {
		logger.Info("* Basic Client ConfigMap compiled")

	}

	if vKafka.Sasl_mechanisms != "" {
		cm["sasl.mechanisms"] = vKafka.Sasl_mechanisms
		cm["security.protocol"] = vKafka.Security_protocol
		cm["sasl.username"] = vKafka.Sasl_username
		cm["sasl.password"] = vKafka.Sasl_password
		if vGeneral.Debuglevel > 0 {
			logger.Info("* Security Authentifaction configured in ConfigMap")

		}
	}

	// Variable p holds the new Producer instance.
	p, err := kafka.NewProducer(&cm)

	// Check for errors in creating the Producer
	if err != nil {
		logger.Error(fmt.Sprintf("😢Oh noes, there's an error creating the Producer! %s", err))

		if ke, ok := err.(kafka.Error); ok == true {
			switch ec := ke.Code(); ec {
			case kafka.ErrInvalidArg:
				logger.Error(fmt.Sprintf("😢 Can't create the producer because you've configured it wrong (code: %d)!\n\t%v\n\nTo see the configuration options, refer to https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md", ec, err))
			default:
				logger.Error(fmt.Sprintf("😢 Can't create the producer (Kafka error code %d)\n\tError: %v\n", ec, err))
			}

		} else {
			// It's not a kafka.Error
			logger.Error(fmt.Sprintf("😢 Oh noes, there's a generic error creating the Producer! %v", err.Error()))
		}
		// call it when you know it's broken
		os.Exit(1)

	}

	if vGeneral.Debuglevel > 0 {
		logger.Info("* Created Kafka Producer instance :")
		logger.Info("")
		logger.Info("**** LETS GO Processing ****")
		logger.Info("")
	}

	///////////////////////////////////////////////////////
	//
	// Successful connection established with Kafka Cluster
	//
	///////////////////////////////////////////////////////

	//
	// For signalling termination from main to go-routine
	termChan := make(chan bool, 1)
	// For signalling that termination is done from go-routine to main
	doneChan := make(chan bool)

	vFlush := 0

	////////////////////////////////////////////////////////////////////////
	// Lets fecth the records that need to be pushed to the fs api end point
	returnedRecs, todo_count := fetchEFTRecords()

	if vGeneral.Eventtype == "paymentRT" {
		info.With(prometheus.Labels{"batch": "rt_rpp"}).Set(float64(todo_count)) // this will be the recordcount of the records returned by the sql query

	} else {
		info.With(prometheus.Labels{"batch": "nrt"}).Set(float64(todo_count)) // this will be the recordcount of the records returned by the sql query

	}

	if vGeneral.Debuglevel > 0 {
		println("Fake Dataset", returnedRecs) // just doing this to prefer a unused error

	}

	// As we're still faking it:
	todo_count = vGeneral.Testsize // this will be recplaced by the value of todo_count from above.

	// now we loop through the results, building a json document based on FS requirements and then post it, for this code I'm posting to
	// Confluent Kafka topic, but it's easy to change to have it post to a API endpoint.

	// this is to keep record of the total run time
	vStart := time.Now()

	for count := 0; count < todo_count; count++ {

		// We're going to time every record and push that to prometheus
		txnStart := time.Now()

		// Eventually we will have a routine here that will not create fake data, but will rather read the data from
		// a file, do some checks and then publish the JSON payload structured Txn to the topic.

		// if 0 then sleep is disabled otherwise
		//
		// lets get a random value 0 -> vGeneral.sleep, then delay/sleep as up to that fraction of a second.
		// this mimics someone thinking, as if this is being done by a human at a keyboard, for batcvh file processing we don't have this.
		// ie if the user said 200 then it implies a randam value from 0 -> 200 milliseconds.

		if vGeneral.Sleep != 0 {
			rand.Seed(time.Now().UnixNano())
			n := rand.Intn(vGeneral.Sleep) // if vGeneral.sleep = 1000, then n will be random value of 0 -> 1000  aka 0 and 1 second
			if vGeneral.Debuglevel >= 2 {
				logger.Info(fmt.Sprintf("Sleeping %d Millisecond...", n))

			}
			time.Sleep(time.Duration(n) * time.Millisecond)
		}

		// Build the entire JSON Payload document from the record fetched
		var cTenant string
		var cMerchant string
		var t_Payment map[string]interface{}
		if vGeneral.Eventtype == "paymentNRT" {
			t_Payment, cTenant, cMerchant = contructPaymentNRTFromFake()

		} else { // paymentRT
			t_Payment, cTenant, cMerchant = contructPaymentRTFromFake()

		}
		cGetRiskStatus := varSeed.Risk[gofakeit.Number(0, 2)]

		// We cheating, going to use this same aggregator in both TPconfigGroups
		t_aggregators := []types.TAggregator{
			types.TAggregator{
				AggregatorId:   "agg1",
				AggregateScore: 0.3,
				MatchedBound:   0.1,
				Alert:          true,
				SuppressAlert:  false,
				Scores: types.TPScore{
					Models: []types.TModels{
						types.TModels{
							ModelId: "model1",
							Score:   0.51,
						},
						types.TModels{
							ModelId: "model2",
							Score:   0.42,
						},
						types.TModels{
							ModelId: "aggModel",
							Score:   0.2,
						},
					},
					Tags: []types.TTags{
						types.TTags{
							Tag:    "codes",
							Values: []string{"Dodgy"},
						},
					},
					Rules: []types.TRules{
						types.TRules{
							RuleId: "rule1",
							Score:  0.2,
						},
						types.TRules{
							RuleId: "rule2",
							Score:  0.4,
						},
					},
				},
				OutputTags: []types.TTags{
					types.TTags{
						Tag:    "codes",
						Values: []string{"LowValueTrans"},
					},
				},
				SuppressedTags: []types.TTags{
					types.TTags{
						Tag:    "otherTag",
						Values: []string{"t3", "t4"},
					},
				},
			},

			types.TAggregator{
				AggregatorId:   "agg2",
				AggregateScore: 0.2,
				MatchedBound:   0.4,
				Alert:          true,
				SuppressAlert:  false,
			},
		}

		t_entity := []types.TPEntity1{
			types.TPEntity1{
				TenantId:   cTenant,    // Tenant = a Bank, our customer
				EntityType: "merchant", // Entity is either a Merchant or the consumer, for outbound is it the consumer, inbound bank its normally the merchant, except when a reversal is actioned.
				EntityId:   cMerchant,  // if Merchant we push a seeded Merchant name into here, not sure how the consumer block would look

				OverallScore: types.TPOverallScore{
					AggregationModel: "aggModel",
					OverallScore:     0.2,
				},

				Models: []types.TModels{
					types.TModels{
						ModelId:    "model1",
						Score:      0.5,
						Confidence: 0.2,
					},
					types.TModels{
						ModelId:    "model2",
						Score:      0.1,
						Confidence: 0.9,
						ModelData: types.TModelData{
							Adata: "aData ASet",
						},
					},
					types.TModels{
						ModelId:    "aggModel",
						Score:      0.2,
						Confidence: 0.9,
						ModelData: types.TModelData{
							Adata: "aData BSet",
						},
					},
				},

				FailedModels: []string{"fm1", "fm2"},

				OutputTags: []types.TTags{
					types.TTags{
						Tag:    "codes",
						Values: []string{"highValueTransaction", "fraud"},
					},
					types.TTags{
						Tag:    "otherTag",
						Values: []string{"t1", "t2"},
					},
				},

				RiskStatus: cGetRiskStatus,

				ConfigGroups: []types.TConfigGroups{
					types.TConfigGroups{
						Type:           "global",
						TriggeredRules: []string{"rule1", "rule2"},
						Aggregators:    t_aggregators,
					},
					types.TConfigGroups{
						Type:           "analytical",
						Id:             "acg1	",
						TriggeredRules: []string{"rule1", "rule3"},
						Aggregators:    t_aggregators,
					},
					types.TConfigGroups{
						Type:           "analytical",
						Id:             "acg2",
						TriggeredRules: []string{"rule1", "rule3"},
					},
					types.TConfigGroups{
						Type:           "tenant",
						Id:             "tenant10",
						TriggeredRules: []string{"rule2", "rule3"},
					},
				},
			},

			types.TPEntity1{
				EntityType: "consumer",
				EntityId:   "consumer1",
			},
		}

		// In real live we'd define the object and then append the array items... of course !!!!
		// we're cheating as we're just creating volume to push onto Kafka and onto MongoDB store.
		t_versions := types.TPVersions{
			ModelGraph: 4,
			ConfigGroups: []types.TConfigGroups{
				types.TConfigGroups{
					Type:    "global",
					Version: "1",
				},
				types.TConfigGroups{
					Type:    "analytical",
					Version: "3",
					Id:      "acg1",
				},
				types.TConfigGroups{
					Type:    "analytical",
					Version: "2",
					Id:      "acg2",
				},
				types.TConfigGroups{
					Type:    "tenant",
					Id:      "tenant10",
					Version: "0",
				},
			},
		}

		t_engineResponse := types.TPEngineResponse{
			Entities:         t_entity,
			JsonVersion:      4,
			OriginatingEvent: t_Payment,
			OutputTime:       time.Now().Format("2006-01-02T15:04:05"),
			ProcessorId:      "proc",
			Versions:         t_versions,
		}

		// Change/Marshal the t_engineResponse variable into an array of bytes required to be send
		valueBytes, err := json.Marshal(t_engineResponse)
		if err != nil {
			logger.Error(fmt.Sprintf("Marchalling error: %s", err))

		}

		kafkaMsg := kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &vKafka.Topicname,
				Partition: kafka.PartitionAny,
			},
			Value: valueBytes,      // This is the payload/body thats being posted
			Key:   []byte(cTenant), // We us this to group the same transactions together in order, IE submitting/Debtor Bank.
		}

		if vGeneral.Debuglevel > 1 && vGeneral.Echojson == 1 {
			prettyJSON(string(valueBytes))

		}

		if vGeneral.PostKafka == 1 {
			// This is where we publish message onto the topic... on the Confluent cluster for now,
			// this will be replaced with a FS API post call

			apiStart := time.Now() // time the api call
			if e := p.Produce(&kafkaMsg, nil); e != nil {
				logger.Error(fmt.Sprintf("😢 Darn, there's an error producing the message! %s", e.Error()))

			}
			//determine the duration of the api call log to prometheus histogram
			rec_duration.WithLabelValues("nrt_eft").Observe(time.Since(apiStart).Seconds())

			//Fush every flush_interval loops
			if vFlush == vKafka.Flush_interval {
				t := 10000
				if r := p.Flush(t); r > 0 {
					logger.Error(fmt.Sprintf("Failed to flush all messages after %d milliseconds. %d message(s) remain", t, r))

				} else {
					if vGeneral.Debuglevel >= 1 {
						logger.Info(fmt.Sprintf("%s/%s, Messages flushed from the queue", count, vFlush))

					}
					vFlush = 0
				}
			}
		}

		if vGeneral.Json_to_file == 1 { // write to Kafka Topic

			eventId := t_engineResponse.OriginatingEvent["eventId"]

			// define, contruct the file name
			loc := fmt.Sprintf("%s/%s.json", vGeneral.Output_path, eventId)
			if vGeneral.Debuglevel > 0 {
				logger.Info(loc)

			}

			//...................................
			// Writing struct type to a JSON file
			//...................................
			// Writing
			// https://www.golangprograms.com/golang-writing-struct-to-json-file.html
			// https://www.developer.com/languages/json-files-golang/
			// Reading
			// https://medium.com/kanoteknologi/better-way-to-read-and-write-json-file-in-golang-9d575b7254f2
			fd, err := json.MarshalIndent(t_engineResponse, "", " ")
			if err != nil {
				logger.Error(fmt.Sprintf("MarshalIndent error %s", err))

			}

			err = ioutil.WriteFile(loc, fd, 0644)
			if err != nil {
				logger.Error(fmt.Sprintf("ioutil.WriteFile error %s", err))

			}
		}

		// We will decide if we want to keep this bit!!! or simplify it.
		//
		// Convenient way to Handle any events (back chatter) that we get
		go func() {
			doTerm := false
			for !doTerm {
				// The `select` blocks until one of the `case` conditions
				// are met - therefore we run it in a Go Routine.
				select {
				case ev := <-p.Events():
					// Look at the type of Event we've received
					switch ev.(type) {

					case *kafka.Message:
						// It's a delivery report
						km := ev.(*kafka.Message)
						if km.TopicPartition.Error != nil {
							logger.Error(fmt.Sprintf("☠️ Failed to send message to topic '%v'\tErr: %v",
								string(*km.TopicPartition.Topic),
								km.TopicPartition.Error))

						} else {
							if vGeneral.Debuglevel > 2 {
								logger.Info(fmt.Sprintf("✅ Message delivered to topic '%v'(partition %d at offset %d)",
									string(*km.TopicPartition.Topic),
									km.TopicPartition.Partition,
									km.TopicPartition.Offset))

							}
						}

					case kafka.Error:
						// It's an error
						em := ev.(kafka.Error)
						logger.Error(fmt.Sprint("☠️ Uh oh, caught an error:\n\t%v", em))

					}
				case <-termChan:
					doTerm = true

				}
			}
			close(doneChan)
		}()

		/////////////////////////////////////////////////////////////////////////////
		// Prometheus Metrics
		//
		// // increment a counter for number of requests processed, we can use this number with time to create a throughput graph
		if vGeneral.Eventtype == "paymentNRT" {
			req_processed.WithLabelValues("nrt").Inc()

			// // determine the duration and log to prometheus histogram, this is for the entire build of payload and post
			api_duration.WithLabelValues("nrt").Observe(time.Since(txnStart).Seconds())

		} else {
			req_processed.WithLabelValues("rt_rpp").Inc()

			// // determine the duration and log to prometheus histogram, this is for the entire build of payload and post
			api_duration.WithLabelValues("rt_rpp").Observe(time.Since(txnStart).Seconds())
		}

		vFlush++

	}

	if vGeneral.Debuglevel > 0 {
		logger.Info("")
		logger.Info("**** DONE Processing ****")
		logger.Info("")
	}

	// --
	// Flush the Producer queue, t = TimeOut, 1000 = 1 second
	t := 10000
	if r := p.Flush(t); r > 0 {
		logger.Error(fmt.Sprintf("Failed to flush all messages after %d milliseconds. %d message(s) remain", t, r))

	} else {
		if vGeneral.Debuglevel >= 1 {
			logger.Info("All messages flushed from the queue")
			logger.Info("")

		}
	}

	if vGeneral.Debuglevel >= 1 {
		vEnd := time.Now()
		logger.Info("", "start", vStart)
		logger.Info("", "end", vEnd)
		logger.Info("", "duration", time.Since(vStart))
		logger.Info("", "records", vGeneral.Testsize)
	}

	// --
	// Stop listening to events and close the producer
	// We're ready to finish
	termChan <- true

	// wait for go-routine to terminate
	<-doneChan
	defer p.Close()

	os.Exit(0)

}

func main() {

	// Initialize Prometheus handler
	logger.Info("**** Starting ****")

	prometheus.MustRegister(sql_duration, api_duration, rec_duration, info)

	go runLoader()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9000", nil)

}
