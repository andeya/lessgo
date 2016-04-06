package lessgo

// 带描述HandlerFunc
type DescHandler struct {
	Handler     HandlerFunc       // 操作
	Description string            // 本操作的描述
	Success     string            // 成功后返回的内容描述
	Failure     string            // 失败后返回的内容描述
	Param       map[string]string // 参数描述
}
