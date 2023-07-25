package dns

import "go.riyazali.net/sqlite"

// ExtensionFunc returns a sqlite.ExtensionFunc that can be used to register dns as a sqlite extension.
func ExtensionFunc() sqlite.ExtensionFunc {
	return func(api *sqlite.ExtensionApi) (_ sqlite.ErrorCode, err error) {
		if err = api.CreateModule("dns", &ResolverModule{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		if err = api.CreateModule("search_list", &SearchList{}); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		// register module's custom function
		type Fn struct {
			Name string
			Func sqlite.Function
		}

		var fns = []Fn{
			{"FQDN", &FQDN{}},
			{"ClassicResolver", &ClassicResolver{}},
			{"TlsResolver", &TlsResolver{}},
			{"SystemResolver", &SystemResolver{}},
		}

		for _, fn := range fns {
			if err = api.CreateFunction(fn.Name, fn.Func); err != nil {
				return sqlite.SQLITE_ERROR, err
			}
		}

		return sqlite.SQLITE_OK, nil
	}
}
