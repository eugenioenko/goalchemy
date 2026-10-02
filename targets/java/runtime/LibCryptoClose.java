package rt;
public final class LibCryptoClose {
 private LibCryptoClose() {}
 public static void libCryptoClose(TaskSpawn.Task t,Native.Key key) {
  try {
   Native.Material material=Native.invalidate(key);
   if(material==null){t.rv=new Object[0];return;}
   Native.async(t,new Object[0],()->{material.close();return new Object[0];},wire->new Object[0]);
  }catch(Native.Reject e){throw new TaskSpawn.HostFault(e.getMessage());}
 }
}
