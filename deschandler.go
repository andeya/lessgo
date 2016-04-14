package lessgo

// 带描述HandlerFunc
type (
	DescHandler struct {
		Handler  func(Context) error // 操作
		Desc     string              // 本操作的描述
		Produces []string            // 支持的响应内容类型，如["application/xml", "application/json"]
		Params   []Param             // 参数说明列表
	}
	Param struct {
		Name     string      // 参数名
		In       string      // 参数出现位置form、query、path、body、header
		Required bool        // 是否必填
		Format   interface{} // 参数值示例(至少为相应go基础类型空值)
		Desc     string      // 参数描述
	}
)
