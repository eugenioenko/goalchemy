package rt

import nativeencoding "github.com/eugenioenko/goalchemy/lib/encoding"

func LibEncodingBase64Encode(data []byte) (string, error) { return nativeencoding.Base64Encode(data) }
