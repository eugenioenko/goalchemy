namespace Rt;
public static partial class R
{
 public const int MAX_FILE=1<<30;
 internal static string osPath(string name,out long status)
 {
  status=0;
  if(name.Length==0){status=1;return null;}
  if(name.IndexOf('\0')>=0){status=8;return null;}
  return System.Text.Encoding.UTF8.GetString(System.Text.Encoding.Latin1.GetBytes(name));
 }
 static bool fileAncestor(string path)
 {
  for(var dir=System.IO.Path.GetDirectoryName(path);!string.IsNullOrEmpty(dir);dir=System.IO.Path.GetDirectoryName(dir))
   if(System.IO.File.Exists(dir))return true;
  return false;
 }
 internal static long osStatus(System.Exception e,string path)
 {
  switch(e){
   case System.IO.FileNotFoundException:return 1;
   case System.IO.DirectoryNotFoundException:return fileAncestor(path)?5:1;
   case System.UnauthorizedAccessException:return System.IO.Directory.Exists(path)?4:3;
   case System.Security.SecurityException:return 3;
   case System.PlatformNotSupportedException:return 7;
   case System.ArgumentException:return 8;
   default:return 9;
  }
 }
 public static object[] libOsReadFile(string name)
 {
  var path=osPath(name,out long bad);
  if(path==null)return new object[]{Slice.BYTE_NIL,bad};
  try{
   if(System.IO.Directory.Exists(path))return new object[]{Slice.BYTE_NIL,4L};
   using var f=System.IO.File.OpenRead(path);
   if(f.Length>MAX_FILE)return new object[]{Slice.BYTE_NIL,6L};
   var data=new byte[f.Length];int n=0;
   while(n<data.Length){int r=f.Read(data,n,data.Length-n);if(r==0)break;n+=r;}
   if(n<data.Length)System.Array.Resize(ref data,n);
   return new object[]{new Slice(data,0,data.Length,data.Length),0L};
  }catch(System.Exception e) when (e is System.IO.IOException||e is System.UnauthorizedAccessException||e is System.Security.SecurityException||e is System.PlatformNotSupportedException||e is System.ArgumentException){return new object[]{Slice.BYTE_NIL,osStatus(e,path)};}
 }
}
