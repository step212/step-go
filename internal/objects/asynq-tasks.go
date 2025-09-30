package objects

const (
	TypeTargetCreate           = "target:create"
	TypeStepCreate             = "step:create"
	TypeStepComment            = "step:comment"
	TypeFeedbackPortraitChange = "feedback:portraitchange"
)

type TargetCreatePayload struct {
	TargetID uint64
}

type StepCreatePayload struct {
	StepID uint64
}

type StepCommentPayload struct {
	StepID      uint64
	CommentType string
}

type PortraitchangeType struct {
	Type  string   // portrait, target
	Scope []string // top dimension for portrait type/top_target_id for target type
}

// 画像、目标得分（目标得分也是画像的一部分）
type FeedbackPortraitChangePayload struct {
	UserID              string
	PortraitChangeTypes []*PortraitchangeType
}

const (
	QueueCritical = "step-go-critical"
	QueueDefault  = "step-go-default"
	QueueLow      = "step-go-low"
)

var Queues = map[string]int{
	QueueCritical: 6,
	QueueDefault:  3,
	QueueLow:      1,
}
