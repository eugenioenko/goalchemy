package rt

import nativeencoding "github.com/eugenioenko/goalchemy/lib/encoding"

func LibEncodingBase64URLEncode(data []byte) (string, error) {
	return nativeencoding.Base64URLEncode(data)
}
