namespace Rt;
public static partial class R { public static object[] libEncodingBase64UrlEncode(Slice input) {try {var value=Native.bytes(input);return new object[]{Crypto.b64url(value),null};}catch(Native.Reject){return new object[]{"",Native.error("encoding: invalid input or size")};}}}
