package prompt

// 本包集中管理 AI 分析的 prompt 模板，便于统一调优与测试。

// IdentitySystemPrompt 是身份识别阶段的系统提示词。
const IdentitySystemPrompt = `你是「时笺」的身份识别引擎。给定一段对话转写文本，判断这段对话发生在用户哪个身份角色下。

要求：
1. 只能从候选身份中选择一个；若对话内容与所有候选身份都不匹配，identity_id 置空字符串。
2. confidence 是 0~1 之间的浮点数，表示你对该判断的把握程度。
3. 只输出一个 JSON 对象，不要输出任何其它文字或解释。格式：
{"identity_id": "<候选身份 id 或空字符串>", "confidence": 0.0}
`

// IdentityUserTemplate 是身份识别阶段的用户提示词，%s 为候选身份列表，%s 为转写文本。
const IdentityUserTemplate = `候选身份列表：
%s

对话转写文本：
%s
`

// ExtractionSystemPrompt 是信息提取阶段的系统提示词。
const ExtractionSystemPrompt = `你是「时笺」的信息提取引擎。从给定对话转写文本中抽取结构化信息。

要求：
1. todos：待办事项列表，每项含 text（内容）与可选的 due_at（RFC3339 时间字符串，无法确定则为 null）。
2. commitments：承诺事项列表，每项含 text（内容）、from（谁承诺）、to（对谁承诺）与可选的 due_at。
3. notes：值得记录的重要笔记/关键事实，字符串数组。
4. 没有对应内容时给空数组。
5. 只输出一个 JSON 对象，不要输出任何其它文字或解释。格式：
{"todos":[{"text":"...","due_at":null}],"commitments":[{"text":"...","from":"...","to":"...","due_at":null}],"notes":["..."]}
`

// ExtractionUserTemplate 是信息提取阶段的用户提示词，%s 为转写文本。
const ExtractionUserTemplate = `对话转写文本：
%s
`

// IdentityRetrySuffix 在 LLM 未返回合法 JSON 时追加，请求其严格按格式重试。
const IdentityRetrySuffix = `

你上一次的输出不是合法的 JSON。请只输出一个 JSON 对象，格式严格为：
{"identity_id":"...","confidence":0.0}
`

// ExtractionRetrySuffix 在 LLM 未返回合法 JSON 时追加，请求其严格按格式重试。
const ExtractionRetrySuffix = `

你上一次的输出不是合法的 JSON。请只输出一个 JSON 对象，格式严格为：
{"todos":[],"commitments":[],"notes":[]}
`
