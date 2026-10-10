package rt;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.*;
public final class LibOsReadFile {
 private LibOsReadFile() {}
 public static final int MAX_FILE=1<<30;
 static Object path(String name){
  if(name.isEmpty())return 1L;
  if(name.indexOf('\0')>=0)return 8L;
  try{return Paths.get(new String(name.getBytes(StandardCharsets.ISO_8859_1),StandardCharsets.UTF_8));}catch(InvalidPathException e){return 8L;}
 }
 static long status(Exception e){
  if(e instanceof NoSuchFileException)return 1;
  if(e instanceof FileAlreadyExistsException)return 2;
  if(e instanceof AccessDeniedException)return 3;
  if(e instanceof NotDirectoryException)return 5;
  if(e instanceof UnsupportedOperationException)return 7;
  String reason=e instanceof FileSystemException f?f.getReason():e.getMessage();
  if(reason==null)return 9;
  switch(reason){
   case "No such file or directory":return 1;
   case "File exists":return 2;
   case "Permission denied":case "Operation not permitted":return 3;
   case "Is a directory":return 4;
   case "Not a directory":return 5;
   case "File too large":return 6;
   case "Invalid argument":return 8;
   default:return 9;
  }
 }
 public static Object[] libOsReadFile(String name){
  Object p=path(name);
  if(p instanceof Long s)return new Object[]{Slice.BYTE_NIL,s};
  try{
   Path path=(Path)p;
   if(Files.isDirectory(path))return new Object[]{Slice.BYTE_NIL,4L};
   if(Files.size(path)>MAX_FILE)return new Object[]{Slice.BYTE_NIL,6L};
   byte[] data=Files.readAllBytes(path);
   if(data.length>MAX_FILE)return new Object[]{Slice.BYTE_NIL,6L};
   return new Object[]{new Slice(data,0,data.length,data.length),0L};
  }catch(IOException|SecurityException|UnsupportedOperationException e){return new Object[]{Slice.BYTE_NIL,e instanceof SecurityException?3L:status(e)};}
 }
}
