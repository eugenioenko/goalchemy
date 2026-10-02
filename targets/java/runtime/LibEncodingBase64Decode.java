package rt;
public final class LibEncodingBase64Decode {
 private LibEncodingBase64Decode() {}
 public static Object[] decode(String input,boolean url) {
  if(input==null||input.length()>((Native.MAX+2)/3)*4)return new Object[]{Slice.BYTE_NIL,Native.error("encoding: invalid input or size")};
  try{
   if(!url&&input.length()%4!=0)throw new IllegalArgumentException();
   int padding=0;
   for(int i=0;i<input.length();i++){char c=input.charAt(i);if(!url&&c=='='){padding++;if(padding>2||i<input.length()-2)throw new IllegalArgumentException();continue;}if(padding!=0||!(c>='A'&&c<='Z'||c>='a'&&c<='z'||c>='0'&&c<='9'||c==(url?'-':'+')||c==(url?'_':'/')))throw new IllegalArgumentException();}
   byte[] bytes=(url?java.util.Base64.getUrlDecoder():java.util.Base64.getDecoder()).decode(input);
   String canonical=(url?java.util.Base64.getUrlEncoder().withoutPadding():java.util.Base64.getEncoder()).encodeToString(bytes);
   if(!canonical.equals(input)||bytes.length>Native.MAX)throw new IllegalArgumentException();
   return new Object[]{Native.slice(bytes),null};
  }catch(IllegalArgumentException e){return new Object[]{Slice.BYTE_NIL,Native.error("encoding: invalid input or size")};}
 }
 public static Object[] libEncodingBase64Decode(String input){return decode(input,false);}
}
