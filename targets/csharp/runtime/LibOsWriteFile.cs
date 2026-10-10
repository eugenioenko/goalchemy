namespace Rt;
public static partial class R
{
 public static long libOsWriteFile(string name,Slice data,long perm)
 {
  var path=osPath(name,out long bad);
  if(path==null)return bad;
  if(data.l>MAX_FILE)return 6;
  try{
   if(System.IO.Directory.Exists(path))return 4;
   var options=new System.IO.FileStreamOptions{Mode=System.IO.FileMode.Create,Access=System.IO.FileAccess.Write};
   if(!System.OperatingSystem.IsWindows())options.UnixCreateMode=(System.IO.UnixFileMode)(perm&0x1ff);
   using var f=new System.IO.FileStream(path,options);
   if(data.l>0)f.Write((byte[])data.a,data.o,data.l);
   return 0;
  }catch(System.Exception e) when (e is System.IO.IOException||e is System.UnauthorizedAccessException||e is System.Security.SecurityException||e is System.PlatformNotSupportedException||e is System.ArgumentException){return osStatus(e,path);}
 }
}
