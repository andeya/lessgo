package lessgo

// 带描述HandlerFunc
type DescHandler struct {
	Handler func(Context) error // 操作
	Desc    string              // 本操作的描述
	Param   map[string]string   // 参数说明
}
