namespace Rt;
public static partial class R { public static object[] libEncodingBase64Encode(Slice input) {try {var value=Native.bytes(input);return new object[]{Convert.ToBase64String(value),null};}catch(Native.Reject){return new object[]{"",Native.error("encoding: invalid input or size")};}}}
