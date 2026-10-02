package rt;
public final class LibCallbackRequest {
 private LibCallbackRequest() {}
 public static void libCallbackRequest(TaskSpawn.Task t,StdContextErr.Context context,String name,Slice input){Callback.request(t,context,name,input);}
}
