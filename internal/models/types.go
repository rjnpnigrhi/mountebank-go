package models

// Request represents a protocol-agnostic request
type Request struct {
	Protocol      string                 `json:"protocol,omitempty"`
	IP            string                 `json:"ip,omitempty"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	
	// HTTP-specific fields
	Method        string                 `json:"method,omitempty"`
	Path          string                 `json:"path,omitempty"`
	Query         map[string]interface{} `json:"query,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	Body          interface{}            `json:"body,omitempty"`
	
	// TCP-specific fields
	Data          string                 `json:"data,omitempty"`
	
	// SMTP-specific fields
	From          string                 `json:"from,omitempty"`
	To            []string               `json:"to,omitempty"`
	Subject       string                 `json:"subject,omitempty"`
	Text          string                 `json:"text,omitempty"`
	HTML          string                 `json:"html,omitempty"`
	
	// Internal fields
	IsDryRun      bool                   `json:"-"`
}

// Response represents a protocol-agnostic response
type Response struct {
	// HTTP-specific fields
	StatusCode    int                    `json:"statusCode,omitempty"`
	Headers       map[string]interface{} `json:"headers,omitempty"`
	Body          interface{}            `json:"body,omitempty"`
	
	// TCP-specific fields
	Data          string                 `json:"data,omitempty"`
	
	// SMTP-specific fields
	Response      string                 `json:"response,omitempty"`
	
	// Proxy-specific fields
	Proxy         interface{}            `json:"proxy,omitempty"`
	CallbackURL   string                 `json:"callbackURL,omitempty"`
	
	// Internal fields
	ProxyResponseTime int                `json:"_proxyResponseTime,omitempty"`
	Blocked           bool               `json:"blocked,omitempty"`
	Code              string             `json:"code,omitempty"`
}

// Predicate represents a request matching condition
type Predicate struct {
	Equals           interface{}            `json:"equals,omitempty"`
	DeepEquals       interface{}            `json:"deepEquals,omitempty"`
	Contains         interface{}            `json:"contains,omitempty"`
	StartsWith       interface{}            `json:"startsWith,omitempty"`
	EndsWith         interface{}            `json:"endsWith,omitempty"`
	Matches          interface{}            `json:"matches,omitempty"`
	Exists           interface{}            `json:"exists,omitempty"`
	Not              *Predicate             `json:"not,omitempty"`
	Or               []Predicate            `json:"or,omitempty"`
	And              []Predicate            `json:"and,omitempty"`
	Inject           string                 `json:"inject,omitempty"`
	
	CaseSensitive    *bool                  `json:"caseSensitive,omitempty"`
	Except           string                 `json:"except,omitempty"`
	XPath            *XPathConfig           `json:"xpath,omitempty"`
	JSONPath         *JSONPathConfig        `json:"jsonpath,omitempty"`
}

// XPathConfig represents XPath selector configuration
type XPathConfig struct {
	Selector string            `json:"selector"`
	NS       map[string]string `json:"ns,omitempty"`
}

// JSONPathConfig represents JSONPath selector configuration
type JSONPathConfig struct {
	Selector string `json:"selector"`
}

// Behavior represents a response transformation
type Behavior struct {
	Wait          *WaitBehavior          `json:"wait,omitempty"`
	Decorate      string                 `json:"decorate,omitempty"`
	Copy          []CopyBehavior         `json:"copy,omitempty"`
	Lookup        *LookupBehavior        `json:"lookup,omitempty"`
	ShellTransform string                `json:"shellTransform,omitempty"`
}

// WaitBehavior represents a wait/latency behavior
type WaitBehavior struct {
	Milliseconds int `json:"milliseconds,omitempty"`
}

// CopyBehavior represents a copy behavior
type CopyBehavior struct {
	From          string        `json:"from"`
	Into          string        `json:"into"`
	Using         *CopySelector `json:"using,omitempty"`
}

// CopySelector represents a selector for copy behavior
type CopySelector struct {
	Method   string                 `json:"method"`
	Selector string                 `json:"selector"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Ns       map[string]string      `json:"ns,omitempty"`
}

// LookupBehavior represents a lookup behavior
type LookupBehavior struct {
	Key           map[string]interface{} `json:"key"`
	FromDataSource *DataSource           `json:"fromDataSource"`
	Into          string                 `json:"into"`
}

// DataSource represents a data source for lookup
type DataSource struct {
	CSV           *CSVDataSource         `json:"csv,omitempty"`
}

// CSVDataSource represents a CSV data source
type CSVDataSource struct {
	Path          string `json:"path"`
	KeyColumn     string `json:"keyColumn"`
	ColumnInto    map[string]string `json:"columnInto,omitempty"`
}

// PredicateGenerator represents a predicate generator for proxy
type PredicateGenerator struct {
	Matches       map[string]interface{} `json:"matches,omitempty"`
	CaseSensitive *bool                  `json:"caseSensitive,omitempty"`
	Except        string                 `json:"except,omitempty"`
	XPath         *XPathConfig           `json:"xpath,omitempty"`
	JSONPath      *JSONPathConfig        `json:"jsonpath,omitempty"`
	Inject        string                 `json:"inject,omitempty"`
	Ignore        map[string]interface{} `json:"ignore,omitempty"`
	PredicateOperator string             `json:"predicateOperator,omitempty"`
}

// ResponseConfig represents a response configuration
type ResponseConfig struct {
	Is            *Response              `json:"is,omitempty"`
	Proxy         *ProxyConfig           `json:"proxy,omitempty"`
	Inject        string                 `json:"inject,omitempty"`
	Fault         *FaultConfig           `json:"fault,omitempty"`
	Behaviors     []Behavior             `json:"behaviors,omitempty"`
	Repeat        int                    `json:"repeat,omitempty"`
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	To                   string                  `json:"to"`
	Mode                 string                  `json:"mode,omitempty"`
	PredicateGenerators  []PredicateGenerator    `json:"predicateGenerators,omitempty"`
	AddWaitBehavior      bool                    `json:"addWaitBehavior,omitempty"`
	AddDecorateBehavior  string                  `json:"addDecorateBehavior,omitempty"`
}

// FaultConfig represents fault injection configuration
type FaultConfig struct {
	Fault string `json:"fault"`
}

// Match represents a debug match entry
type Match struct {
	Timestamp      string          `json:"timestamp"`
	Request        *Request        `json:"request"`
	Response       *Response       `json:"response"`
	ResponseConfig *ResponseConfig `json:"responseConfig"`
	Duration       int             `json:"duration"`
}

// Stub represents a stub with predicates and responses
type Stub struct {
	Predicates []Predicate      `json:"predicates,omitempty"`
	Responses  []ResponseConfig `json:"responses"`
	Matches    []Match          `json:"matches,omitempty"`
	
	// Internal
	IsProxy    bool             `json:"-"`
}

// ImposterConfig represents the configuration for creating an imposter
type ImposterConfig struct {
	Protocol          string                 `json:"protocol"`
	Port              int                    `json:"port,omitempty"`
	Name              string                 `json:"name,omitempty"`
	RecordRequests    bool                   `json:"recordRequests,omitempty"`
	Stubs             []Stub                 `json:"stubs,omitempty"`
	DefaultResponse   *Response              `json:"defaultResponse,omitempty"`
	AllowCORS         bool                   `json:"allowCORS,omitempty"`
	Middleware        string                 `json:"middleware,omitempty"`
	
	// HTTP-specific
	Key               string                 `json:"key,omitempty"`
	Cert              string                 `json:"cert,omitempty"`
	MutualAuth        bool                   `json:"mutualAuth,omitempty"`
	
	// TCP-specific
	Mode              string                 `json:"mode,omitempty"`
	
	// Common
	Host              string                 `json:"host,omitempty"`
}
