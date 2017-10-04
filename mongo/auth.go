package mongo

type AuthCmd struct {
	Authenticate int

	Nonce string
	User  string
	Key   string
}

type StartSaslCmd struct {
	StartSASL int `bson:"startSasl"`
}

type AuthResult struct {
	ErrMsg string
	Ok     bool
}

type GetNonceCmd struct {
	GetNonce int
}

type GetNonceResult struct {
	Nonce string
	Err   string "$err"
	Code  int
}

type LogoutCmd struct {
	Logout int
}

type SaslCmd struct {
	Start          int    `bson:"saslStart,omitempty"`
	Continue       int    `bson:"saslContinue,omitempty"`
	ConversationId int    `bson:"conversationId,omitempty"`
	Mechanism      string `bson:"mechanism,omitempty"`
	Payload        []byte
}

type SaslResult struct {
	Ok    bool `bson:"ok"`
	NotOk bool `bson:"code"` // Server <= 2.3.2 returns ok=1 & code>0 on errors (WTF?)
	Done  bool

	ConversationId int `bson:"conversationId"`
	Payload        []byte
	ErrMsg         string
}

type InsertOp struct {
	collection string        // "database.collection"
	documents  []interface{} // One or more documents to insert
	flags      uint32
}

type InsertQuery struct {
	Collection string
	Documents  []interface{}
	Ordered    bool
	// WriteConcern             interface{}
	BypassDocumentValidation bool
}
