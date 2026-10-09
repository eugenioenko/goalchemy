namespace Rt;
public static partial class R {public static void libLogEmit(long level,long unixNano,string message,Slice attrs,string text)=>Log.Emit(level,unixNano,message,attrs,text);}
