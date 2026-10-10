package rt;
import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.channels.SeekableByteChannel;
import java.nio.file.*;
import java.nio.file.attribute.PosixFilePermission;
import java.nio.file.attribute.PosixFilePermissions;
import java.util.EnumSet;
import java.util.Set;
public final class LibOsWriteFile {
 private LibOsWriteFile() {}
 private static final PosixFilePermission[] BITS={PosixFilePermission.OTHERS_EXECUTE,PosixFilePermission.OTHERS_WRITE,PosixFilePermission.OTHERS_READ,
  PosixFilePermission.GROUP_EXECUTE,PosixFilePermission.GROUP_WRITE,PosixFilePermission.GROUP_READ,
  PosixFilePermission.OWNER_EXECUTE,PosixFilePermission.OWNER_WRITE,PosixFilePermission.OWNER_READ};
 private static SeekableByteChannel open(Path path,Set<OpenOption> options,Set<PosixFilePermission> mode)throws IOException{
  try{return Files.newByteChannel(path,options,PosixFilePermissions.asFileAttribute(mode));}
  catch(UnsupportedOperationException e){return Files.newByteChannel(path,options);}
 }
 public static long libOsWriteFile(String name,Slice data,long perm){
  Object p=LibOsReadFile.path(name);
  if(p instanceof Long s)return s;
  if(data.l>LibOsReadFile.MAX_FILE)return 6;
  byte[] body=data.a==null?new byte[0]:java.util.Arrays.copyOfRange((byte[])data.a,data.o,data.o+data.l);
  Path path=(Path)p;
  Set<OpenOption> options=Set.of(StandardOpenOption.WRITE,StandardOpenOption.CREATE,StandardOpenOption.TRUNCATE_EXISTING);
  Set<PosixFilePermission> mode=EnumSet.noneOf(PosixFilePermission.class);
  for(int i=0;i<9;i++)if((perm>>i&1)!=0)mode.add(BITS[i]);
  try{
   try(SeekableByteChannel ch=open(path,options,mode)){ByteBuffer b=ByteBuffer.wrap(body);while(b.hasRemaining())ch.write(b);}
   return 0;
  }catch(IOException|SecurityException e){return e instanceof SecurityException?3:LibOsReadFile.status(e);}
 }
}
