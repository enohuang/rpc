package micro

type UserService interface {
	GetByID(int int)
}

// 假设这是代码生成
type UserServiceGen struct {
}

// 利用ast抽象语法树
func (u *UserServiceGen) GetByID(id int) {
	// 这段代码是生成的
	req := &Request{ServiceName: "UserService", MethodName: "GetByID", Args: []any{id}}
	//接下来就是rpc 核心
	_ = req

}

type Request struct {
	ServiceName string
	MethodName  string
	Args        []any
}
