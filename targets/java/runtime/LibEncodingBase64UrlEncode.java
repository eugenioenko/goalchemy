package rt;
public final class LibEncodingBase64UrlEncode {
 private LibEncodingBase64UrlEncode() {}
 public static Object[] libEncodingBase64UrlEncode(Slice value) {
  try{return new Object[]{java.util.Base64.getUrlEncoder().withoutPadding().encodeToString(Native.bytes(value)),null};}
  catch(Native.Reject e){return new Object[]{"",Native.error("encoding: invalid input or size")};}
 }
}
