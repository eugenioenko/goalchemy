package rt

import nativeencoding "github.com/eugenioenko/goalchemy/lib/encoding"

func LibEncodingBase64Decode(data string) ([]byte, error) { return nativeencoding.Base64Decode(data) }
