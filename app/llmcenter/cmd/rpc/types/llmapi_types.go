package types

// LLMApiRequest 是调用星火大模型聊天接口的请求体结构
type LLMApiRequest struct {
	FlowID     string        `json:"flow_id"`
	UID        string        `json:"uid"`
	Parameters LLMParameters `json:"parameters"`
	Ext        LLMExt        `json:"ext"`
	Stream     bool          `json:"stream"`
	ChatID     string        `json:"chat_id,omitempty"`
	History    []LLMMessage  `json:"history,omitempty"`
}

type LLMParameters struct {
	AgentUserInput string `json:"AGENT_USER_INPUT"`
	Img            string `json:"img,omitempty"`
}

type LLMExt struct {
	BotID  string `json:"bot_id"`
	Caller string `json:"caller"`
}

type LLMMessage struct {
	Role        string `json:"role"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

// LLMApiResponse 是解析星火 API 流式响应的结构
type LLMApiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	EventData *LLMEventData `json:"event_data,omitempty"`
}

// LLMEventData 封装了中断事件
type LLMEventData struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"` // "interrupt"
	Value     struct {
		Type    string `json:"type"`    // "direct" or "option"
		Content string `json:"content"` // 问题内容
	} `json:"value"`
}

type LLMResumeApiRequest struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"` // "resume", "ignore", "abort"
	Content   string `json:"content"`
}
