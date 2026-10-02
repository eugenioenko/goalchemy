namespace Rt;

public sealed class SelectState
{
    internal bool done;
}

public sealed class Waiter
{
    internal GoTask task;
    internal object val;
    internal bool active = true;
    internal readonly SelectState sel;
    internal readonly int idx;

    internal Waiter(GoTask task, object val, SelectState sel, int idx)
    {
        this.task = task;
        this.val = val;
        this.sel = sel;
        this.idx = idx;
    }

    internal bool live() => active && task != null && task.owner.live(task) && (sel == null || !sel.done);
    internal void clear() { active = false; val = null; task = null; }

    internal void recvDone(object v, bool ok)
    {
        if (!live()) return;
        task.owner.assertDriver(); active = false;
        if (sel != null)
        {
            sel.done = true;
            task.rv = new object[] { (long)idx, v, ok };
        }
        else
        {
            task.rv = new object[] { v, ok };
        }
        task.owner.ready(task);
    }

    internal void sendDone(bool closed)
    {
        if (!live()) return;
        task.owner.assertDriver(); active = false;
        if (sel != null)
        {
            sel.done = true;
            task.rv = new object[] { (long)idx, null, false };
        }
        else
        {
            task.rv = Array.Empty<object>();
        }
        if (closed) task.resumePanic = Panics.plainPanic("send on closed channel");
        task.owner.ready(task);
    }
}

public sealed class Chan
{
    public readonly Queue<object> buf = new();
    public readonly int size;
    public bool closed;
    internal List<Waiter> recvq = new();
    internal List<Waiter> sendq = new();
    internal readonly Func<object> zero;

    internal Chan(int size, Func<object> zero)
    {
        this.size = size;
        this.zero = zero;
    }
}

/// <summary>core.chan.make: channels with a buffer and FIFO wait queues.</summary>
public static partial class R
{
    internal static Waiter dequeue(List<Waiter> q)
    {
        while (q.Count > 0)
        {
            var w = q[0];
            q.RemoveAt(0);
            if (w.live()) return w;
            w.clear();
        }
        return null;
    }

    internal static bool hasLive(List<Waiter> q)
    {
        foreach (var w in q) if (w.live()) return true;
        return false;
    }

    /// <summary>Receives without blocking: {value, ok, done}.</summary>
    internal static object[] tryRecv(Chan ch)
    {
        if (ch.buf.Count > 0)
        {
            var v = ch.buf.Dequeue();
            var w = dequeue(ch.sendq);
            if (w != null)
            {
                ch.buf.Enqueue(w.val);
                w.sendDone(false);
            }
            return new object[] { v, true, true };
        }
        var s = dequeue(ch.sendq);
        if (s != null)
        {
            s.sendDone(false);
            return new object[] { s.val, true, true };
        }
        if (ch.closed) return new object[] { ch.zero(), false, true };
        return new object[] { null, false, false };
    }

    public static Chan makeChan(long size, Func<object> zero)
    {
        if (size < 0 || size > (1L << 53)) throw Panics.plainPanic("makechan: size out of range");
        return new Chan((int)size, zero);
    }
}
