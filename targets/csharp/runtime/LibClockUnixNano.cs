namespace Rt;
public static partial class R {public static long libClockUnixNano()=>checked((System.DateTimeOffset.UtcNow.UtcTicks-System.DateTimeOffset.UnixEpoch.UtcTicks)*100);}
