package types

// Structs - the values we bring in from *app.json configuration file
type TGeneral struct {
	EchoGeneralConfig   int
	Hostname            string
	Loglevel            string
	Debuglevel          int
	Echojson            int
	PostKafka           int
	Testsize            int     // Used to limit number of records posted, over rided when reading test cases from input_source,
	Sleep               int     // sleep time between API post
	Eventtype           string  // paymentRT or paymentNRT
	Json_to_file        int     // Do we output JSON to file in output_path
	Output_path         string  // output location
	Json_from_file      int     // Do we read JSON from input_path directory and post to FS API endpoint
	Input_path          string  // Where are my scenario JSON files located
	MinTransactionValue float64 // Min value if the fake transaction
	MaxTransactionValue float64 // Max value of the fake transaction
	SeedFile            string  // Which seed file to read in
	EchoSeed            int     // 0/1 Echo the seed data to terminal
	IntroBadEntity      int
	IntroBadPayer       int
	IntroBadAgent       int
}

type TKafka struct {
	EchoKafkaConfig   int
	Bootstrapservers  string
	Topicname         string
	Numpartitions     int
	Replicationfactor int
	Retension         string
	Parseduration     string
	Security_protocol string
	Sasl_mechanisms   string
	Sasl_username     string
	Sasl_password     string
	Flush_interval    int
}

// My fake data structures - GoFakeIt
type TAddress struct {
	AddressLine1          string `json:"addressLine1,omitempty"`
	AddressLine2          string `json:"addressLine2,omitempty"`
	AddressLine3          string `json:"addressLine3,omitempty"`
	AddressType           string `json:"addressType,omitempty"`
	Country               string `json:"country,omitempty"`
	CountrySubDivision    string `json:"countrySubDivision,omitempty"`
	Latitude              string `json:"latitude,omitempty"`
	Longitude             string `json:"longitude,omitempty"`
	PostalCode            string `json:"postalCode,omitempty"`
	TownName              string `json:"townName,omitempty"`
	FullAddress           string `json:"fullAddress,omitempty"`
	ResidentAtAddressFrom string `json:"residentAtAddressFrom,omitempty"`
	ResidentAtAddressTo   string `json:"residentAtAddressTo,omitempty"`
	TimeAtAddress         string `json:"timeAtAddress,omitempty"`
}

type TCcard struct {
	Type   string `json:"type"`
	Number int    `json:"number"`
	Exp    string `json:"exp"`
	Cvv    string `json:"cvv"`
}

type TJobinfo struct {
	Company    string `json:"company"`
	Title      string `json:"title"`
	Descriptor string `json:"descriptor"`
	Level      string `json:"level"`
}

type TContact struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type TPerson struct {
	Ssn          string   `json:"ssn"`
	Firstname    string   `json:"firstname"`
	Lastname     string   `json:"lastname"`
	Gender       string   `json:"gender"`
	Address      TAddress `json:"address"`
	Contact      TContact `json:"contact"`
	Ccard        TCcard   `json:"ccard"`
	Job          TJobinfo `json:"job"`
	Created_date string   `json:"created_date"`
}

// FS engineResponse components
type TAmount struct {
	BaseCurrency     string  `json:"baseCurrency,omitempty"`
	BaseValue        float64 `json:"baseValue,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	Value            float64 `json:"value,omitempty"`
	MerchantCurrency string  `json:"merchantCurrency,omitempty"`
	MerchantValue    float64 `json:"merchantValue,omitempty"`
}

type TTags struct {
	Tag    string   `json:"tag,omitempty"`
	Values []string `json:"values,omitempty"`
}

type TModelData struct {
	Adata string `json:"adata,omitempty"`
}

type TModels struct {
	ModelId    string     `json:"modelid,omitempty"`
	Score      float64    `json:"score,omitempty"`
	Confidence float64    `json:"confidence,omitempty"`
	Tags       []TTags    `json:"tags,omitempty"`
	ModelData  TModelData `json:"modeldata,omitempty"`
}

type TRules struct {
	RuleId string  `json:"ruleid,omitempty"`
	Score  float64 `json:"score,omitempty"`
}

type TPScore struct {
	Models []TModels `json:"models,omitempty"`
	Tags   []TTags   `json:"tags,omitempty"`
	Rules  []TRules  `json:"rules,omitempty"`
}

type TAggregator struct {
	AggregatorId   string  `json:"aggregatorId,omitempty"`
	Scores         TPScore `json:"scores,omitempty"`
	AggregateScore float64 `json:"aggregateScore,omitempty"`
	MatchedBound   float64 `json:"matchedBound,omitempty"`
	OutputTags     []TTags `json:"outputTags,omitempty"`
	SuppressedTags []TTags `json:"suppressedTags,omitempty"`
	Alert          bool    `json:"alert,omitempty"`
	SuppressAlert  bool    `json:"suppressAlert,omitempty"`
}

type TConfigGroups struct {
	Type           string        `json:"type,omitempty"`
	Version        string        `json:"version,omitempty"`
	Id             string        `json:"id,omitempty"`
	TriggeredRules []string      `json:"triggeredRules,omitempty"`
	Aggregators    []TAggregator `json:"aggregators,omitempty"`
}

type TPVersions struct {
	ModelGraph   int             `json:"modelgraph,omitempty"`
	ConfigGroups []TConfigGroups `json:"configGroups,omitempty"`
}

type TPOverallScore struct {
	AggregationModel string  `json:"aggregationModel,omitempty"`
	OverallScore     float64 `json:"overallScore,omitempty"`
}

type TPEntity1 struct {
	TenantId     string          `json:"tenantId,omitempty"`
	EntityType   string          `json:"entityType,omitempty"`
	EntityId     string          `json:"entityId,omitempty"`
	OverallScore TPOverallScore  `json:"overallScore,omitempty"`
	Models       []TModels       `json:"models,omitempty"`
	FailedModels []string        `json:"failedModels,omitempty"`
	OutputTags   []TTags         `json:"outputTags,omitempty"`
	RiskStatus   string          `json:"riskStatus,omitempty"`
	ConfigGroups []TConfigGroups `json:"configGroups,omitempty"`
}

type TPEngineResponse struct {
	Entities         []TPEntity1            `json:"entities,omitempty"`
	JsonVersion      int                    `json:"jsonVersion,omitempty"`
	OriginatingEvent map[string]interface{} `json:"originatingEvent,omitempty"`
	OutputTime       string                 `json:"outputTime,omitempty"`
	ProcessorId      string                 `json:"processorId,omitempty"`
	Versions         TPVersions             `json:"versions,omitempty"`
}

type TName struct {
	FullName     string `json:"fullName,omitempty"`
	GivenName    string `json:"givenName,omitempty"`
	MiddleName   string `json:"middleName,omitempty"`
	NamePrefix   string `json:"namePrefix,omitempty"`
	PreviousName string `json:"previousName,omitempty"`
	Surname      string `json:"surname,omitempty"`
}

type TDecoration struct {
	Decoration []string `json:"decoration,omitempty"`
}

type TTenant struct {
	Name     string `json:"name,omitempty"`
	TenantId string `json:"tenantid,omitempty"`
}

type TEntity2 struct {
	Id            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	EntityId      string `json:"entityid,omitempty"`
	TenantId      string `json:"tenantid,omitempty"`
	AccountNumber string `json:"accountnumber,omitempty"`
}

type TPayer struct {
	Id            string `json:"id,omitempty"`
	Name          TName  `json:"name,omitempty"`
	TenantId      string `json:"tenantid,omitempty"`
	AccountNumber string `json:"accountnumber,omitempty"`
}

type TAgent struct {
	Name      string   `json:"name,omitempty"`
	Id        string   `json:"id,omitempty"`
	Address   TAddress `json:"address,omitempty"`
	AccountId string   `json:"accountId,omitempty"`
}

type TPSeed struct {
	Decoration               []string   `json:"decoration,omitempty"`
	Direction                []string   `json:"direction,omitempty"`
	TransactionTypesNrt      []string   `json:"transactionTypesNrt,omitempty"`
	TransactionTypesRt       []string   `json:"transactionTypesRt,omitempty"`
	ChargeBearers            []string   `json:"chargeBearers,omitempty"`
	RemittanceLocationMethod []string   `json:"remittanceLocationMethod,omitempty"`
	SettlementMethod         []string   `json:"settlementMethod,omitempty"`
	VerificationResult       []string   `json:"verificationResult,omitempty"`
	PaymentFrequency         []string   `json:"paymentFrequency,omitempty"`
	Risk                     []string   `json:"risk,omitempty"`
	GoodAgent                []TAgent   `json:"goodAgent,omitempty"`
	BadAgent                 []TAgent   `json:"badAgent,omitempty"`
	Tenants                  []TTenant  `json:"tenants,omitempty"`
	GoodEntities             []TEntity2 `json:"goodEntities,omitempty"`
	BadEntities              []TEntity2 `json:"badEntities,omitempty"`
	GoodPayers               []TPayer   `json:"goodPayers,omitempty"`
	BadPayers                []TPayer   `json:"badPayers,omitempty"`
}

type TVerificationType struct {
	Aa                           string `json:"aa,omitempty"`
	AccountDigitalSignature      string `json:"accountDigitalSignature,omitempty"`
	AuthenticationToken          string `json:"authenticationToken,omitempty"`
	Avs                          string `json:"avs,omitempty"`
	Biometry                     string `json:"biometry,omitempty"`
	CardholderIdentificationData string `json:"cardholderIdentificationData,omitempty"`
	CryptogramVerification       string `json:"cryptogramVerification,omitempty"`
	CscVerification              string `json:"cscVerification,omitempty"`
	Cvv                          string `json:"cvv,omitempty"`
	OfflinePIN                   string `json:"offlinePIN,omitempty"`
	OneTimePassword              string `json:"oneTimePassword,omitempty"`
	OnlinePIN                    string `json:"onlinePIN,omitempty"`
	Other                        string `json:"other,omitempty"`
	PaperSignature               string `json:"paperSignature,omitempty"`
	PassiveAuthentication        string `json:"passiveAuthentication,omitempty"`
	Password                     string `json:"password,omitempty"`
	ThreeDS                      string `json:"threeDS,omitempty"`
	TokenAuthentication          string `json:"tokenAuthentication,omitempty"`
}

type TDevice struct {
	AnonymizerInUseFlag    string  `json:"anonymizerInUseFlag,omitempty"`
	AreaCode               string  `json:"areaCode,omitempty"`
	BrowserType            string  `json:"browserType,omitempty"`
	BrowserVersion         string  `json:"browserVersion"`
	City                   string  `json:"city,omitempty"`
	ClientTimezone         string  `json:"clientTimezone,omitempty"`
	ContinentCode          string  `json:"continentCode,omitempty"`
	CookieId               string  `json:"cookieId"`
	CountryCode            string  `json:"countryCode,omitempty"`
	CountryName            string  `json:"countryName,omitempty"`
	DeviceFingerprint      string  `json:"deviceFingerprint,omitempty"`
	DeviceIMEI             string  `json:"deviceIMEI,omitempty"`
	DeviceName             string  `json:"deviceName,omitempty"`
	FlashPluginPresent     string  `json:"flashPluginPresent,omitempty"`
	HttpHeader             string  `json:"httpHeader,omitempty"`
	IpAddress              string  `json:"ipAddress,omitempty"`
	IpAddressV4            string  `json:"ipAddressV4,omitempty"`
	IpAddressV6            string  `json:"ipAddressV6,omitempty"`
	MetroCode              string  `json:"metroCode,omitempty"`
	MimeTypesPresent       string  `json:"mimeTypesPresent,omitempty"`
	MobileNumberDeviceLink string  `json:"mobileNumberDeviceLink,omitempty"`
	NetworkCarrier         string  `json:"networkCarrier,omitempty"`
	OS                     string  `json:"oS,omitempty"`
	PostalCode             string  `json:"postalCode,omitempty"`
	ProxyDescription       string  `json:"proxyDescription,omitempty"`
	ProxyType              string  `json:"proxyType,omitempty"`
	Region                 string  `json:"region,omitempty"`
	ScreenResolution       string  `json:"screenResolution,omitempty"`
	SessionLatitude        float64 `json:"sessionLatitude,omitempty"`
	SessionLongitude       float64 `json:"sessionLongitude,omitempty"`
	Timestamp              string  `json:"timestamp,omitempty"`
	Type                   string  `json:"type,omitempty"`
	UserAgentString        string  `json:"userAgentString,omitempty"`
}
