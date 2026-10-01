namespace Rt;

/// <summary>std.sync.waitgroup.done.</summary>
public static partial class R
{
    public static void stdSyncWaitgroupDone(WaitGroup wg) => stdSyncWaitgroupAdd(wg, -1);
}
