package rt;
public final class LibLogEmit {
 private LibLogEmit() {}
 public static void libLogEmit(long level,long unixNano,String message,Slice attrs,String text){Log.emit(level,unixNano,message,attrs,text);}
}
