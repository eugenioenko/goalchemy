package rt

import nativeencoding "github.com/eugenioenko/goalchemy/lib/encoding"

func LibEncodingBase64URLDecode(data string) ([]byte, error) {
	return nativeencoding.Base64URLDecode(data)
}
