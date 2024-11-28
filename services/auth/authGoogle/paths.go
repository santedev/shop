package authGoogle

import "shop/views/layouts"

func JsSource() []layouts.Source {
	return []layouts.Source{
		{
			Path: layouts.Scripts+"/googleOAuth.js",
			Kind: layouts.Script{},
			Priority: layouts.High,
		},
		{
			Path:     "https://accounts.google.com/gsi/client",
			Kind:     layouts.Script{Async: true},
			Priority: layouts.Low,
		},
	}
}
