package foobarbaz

import "github.com/google/wire"

// SuperSet 大集合
var SuperSet = wire.NewSet(ProvideFoo, ProvideBar, ProvideBaz)
