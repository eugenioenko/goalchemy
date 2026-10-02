namespace Rt;
public static partial class R
{
 public static void libCryptoClose(GoTask task,Native.Key key)
 { try {var material=Native.invalidate(key);if(material==null){task.rv=Array.Empty<object>();return;}Native.async(task,Array.Empty<object>(),()=>{material.close();return Array.Empty<object>();},wire=>Array.Empty<object>());}catch(Native.Reject e){throw new HostFault(e.Message);} }
}
