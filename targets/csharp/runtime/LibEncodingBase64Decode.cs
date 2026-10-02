namespace Rt;
public static partial class R
{
 public static object[] encodingDecode(string input,bool url)
 {
  try{if(input==null||input.Length>((Native.MAX+2)/3)*4)throw new FormatException();if(!url&&input.Length%4!=0)throw new FormatException();
   foreach(char c in input)if(!(c>='A'&&c<='Z'||c>='a'&&c<='z'||c>='0'&&c<='9'||c==(url?'-':'+')||c==(url?'_':'/')||!url&&c=='='))throw new FormatException();
   string padded=url?input.Replace('-','+').Replace('_','/')+new string('=',(4-input.Length%4)%4):input;var value=Convert.FromBase64String(padded);
   if((url?Crypto.b64url(value):Convert.ToBase64String(value))!=input||value.Length>Native.MAX)throw new FormatException();return new object[]{Native.slice(value),null};
  }catch(FormatException){return new object[]{Slice.BYTE_NIL,Native.error("encoding: invalid input or size")};}
 }
 public static object[] libEncodingBase64Decode(string input)=>encodingDecode(input,false);
}
