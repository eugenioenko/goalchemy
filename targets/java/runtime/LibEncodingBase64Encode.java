package rt;
public final class LibEncodingBase64Encode {
 private LibEncodingBase64Encode() {}
 public static Object[] libEncodingBase64Encode(Slice value) {
  try{return new Object[]{java.util.Base64.getEncoder().encodeToString(Native.bytes(value)),null};}
  catch(Native.Reject e){return new Object[]{"",Native.error("encoding: invalid input or size")};}
 }
}
