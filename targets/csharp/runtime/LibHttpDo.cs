using System.Net.Http;
using System.IO;
namespace Rt;

/// Dedicated bounded transport. Stream, client and cancellation work release before ACK.
public static partial class R
{
    static readonly object[] HTTP_ZERO={0L,Slice.NIL,Slice.BYTE_NIL,null};
    static readonly HashSet<string> HTTP_FORBIDDEN=new(StringComparer.OrdinalIgnoreCase){"Host","Content-Length","Transfer-Encoding","Connection","Proxy-Authorization","Proxy-Connection","Upgrade","Trailer","TE"};
    static bool httpToken(string value){if(value.Length==0)return false;foreach(char c in value)if(!(c>='A'&&c<='Z'||c>='a'&&c<='z'||c>='0'&&c<='9'||"!#$%&'*+-.^_`|~".Contains(c)))return false;return true;}
    static string httpCanonical(string value){var chars=value.ToCharArray();bool upper=true;for(int i=0;i<chars.Length;i++){chars[i]=upper?char.ToUpperInvariant(chars[i]):char.ToLowerInvariant(chars[i]);upper=chars[i]=='-';}return new string(chars);}
    internal static Func<Stream,Stream> httpBodyWrapper=value=>value;
    sealed class HttpLease
    {
        internal readonly System.Threading.CancellationTokenSource cancel=new();readonly List<System.Threading.Tasks.Task> stoppers=new();bool stopped,sealedCleanup;Exception fault;
        internal void Stop(){lock(this){if(stopped)return;stopped=true;if(!sealedCleanup)try{stoppers.Add(cancel.CancelAsync());}catch(Exception e){fault=e;}}}
        internal void Cleanup(){System.Threading.Tasks.Task[] tasks;lock(this){sealedCleanup=true;tasks=stoppers.ToArray();}Exception failure=null;foreach(var task in tasks)try{task.GetAwaiter().GetResult();}catch(Exception e){failure=e;}try{cancel.Dispose();}catch(Exception e){failure=e;}lock(this){stoppers.Clear();failure=fault??failure;}if(failure!=null)throw new HostFault("HTTP cancellation cleanup: "+failure);}
    }
    public static void libHttpDo(GoTask task,GoContext context,string method,string raw,Slice headers,Slice input,long max,long timeoutMillis)
    {
        const string invalid="http: invalid request or limit";
        if(context==null||context.owner!=null&&context.owner!=sched||method!="GET"&&method!="POST"||max<0||max>Native.MAX||timeoutMillis<1||timeoutMillis>300000||raw.Length>8192){task.rv=Native.failure(HTTP_ZERO,invalid);return;}
        observe(context);if(context.err!=null){task.rv=new object[]{0L,Slice.NIL,Slice.BYTE_NIL,context.err};return;}
        var lease=new HttpLease();var boundary=new HostBoundary(context,timeoutMillis*1000000,lease.Stop);HttpRequestMessage request=null;bool gzip=false;
        try
        {
            byte[] body=Native.bytes(input);string[] pairs=Native.strings(headers);if(method=="GET"&&body.Length!=0||pairs.Length%2!=0||pairs.Length>32768)throw new Native.Reject(invalid);
            string url=Native.utf8(raw);foreach(char c in url)if(c<=32||c==127||c=='\ufeff')throw new Native.Reject(invalid);
            if(!Uri.TryCreate(url,UriKind.Absolute,out var uri)||uri.Scheme!="http"&&uri.Scheme!="https"||string.IsNullOrEmpty(uri.Host)||uri.UserInfo.Length!=0||uri.Fragment.Length!=0||uri.Port<1||uri.Port>65535||url.Contains('\\'))throw new Native.Reject(invalid);
            request=new HttpRequestMessage(new HttpMethod(method),uri){Version=System.Net.HttpVersion.Version11,VersionPolicy=HttpVersionPolicy.RequestVersionExact};
            if(method=="POST"||body.Length!=0)request.Content=new ByteArrayContent(body);int total=0;bool encoded=false;
            for(int i=0;i<pairs.Length;i+=2)
            {
                string name=pairs[i],value=pairs[i+1];total+=name.Length+value.Length+4;if(total>65536||!httpToken(name)||HTTP_FORBIDDEN.Contains(name))throw new Native.Reject(invalid);
                foreach(char c in value)if(c>255||c==127||c<32&&c!='\t')throw new Native.Reject(invalid);
                if(!request.Headers.TryAddWithoutValidation(name,value)){request.Content??=new ByteArrayContent(body);if(!request.Content.Headers.TryAddWithoutValidation(name,value))throw new Native.Reject(invalid);}
                if(name.Equals("Accept-Encoding",StringComparison.OrdinalIgnoreCase))encoded=true;
            }
            gzip=!encoded;if(gzip)request.Headers.TryAddWithoutValidation("Accept-Encoding","gzip");
        }
        catch(Native.Reject){request?.Dispose();boundary.Dispose();lease.Cleanup();task.rv=Native.failure(HTTP_ZERO,invalid);return;}
        if(boundary.error()!=null){request.Dispose();boundary.Dispose();lease.Cleanup();task.rv=new object[]{0L,Slice.NIL,Slice.BYTE_NIL,context.err??CONTEXT_DEADLINE_EXCEEDED};return;}
        var token=sched.registerHost(task,lease.Stop,boundary.Dispose,()=>boundary.error()==null?null:new object[]{0L,Slice.NIL,Slice.BYTE_NIL,boundary.error()},
            values=>values[0]!=null?Native.failure(HTTP_ZERO,(string)values[0]):new object[]{(long)values[1],Native.strings(Array.ConvertAll((object[])values[2],v=>(string)v)),Native.slice((byte[])values[3]),null});
        var work=new HttpWork(lease,token,request,gzip,(int)max);System.Threading.Thread worker;
        try{worker=new System.Threading.Thread(work.Run){IsBackground=true,Name="goalchemy-http"};worker.Start();}
        catch(Exception e){Exception fault=e;try{request.Dispose();}catch(Exception cleanup){fault=cleanup;}try{lease.Cleanup();}catch(Exception cleanup){fault=cleanup;}token.complete(Array.Empty<object>(),new HostFault("HTTP submission/cleanup: "+fault));token.acknowledgeCleanup();return;}
        Native.acknowledgeAfter(worker,token);
    }
    sealed class HttpWork
    {
        readonly HttpLease lease;readonly HostToken token;HttpRequestMessage request;readonly bool gzip;readonly int limit;
        internal HttpWork(HttpLease lease,HostToken token,HttpRequestMessage request,bool gzip,int limit){this.lease=lease;this.token=token;this.request=request;this.gzip=gzip;this.limit=limit;}
        internal void Run()
        {
            HttpClient client=null;SocketsHttpHandler handler=null;HttpResponseMessage response=null;var bodies=new List<Stream>();Exception readFailure=null;HostFault fault=null;object[] result=null;string rejection=null;
            try
            {
                handler=new SocketsHttpHandler{AllowAutoRedirect=false,AutomaticDecompression=System.Net.DecompressionMethods.None,MaxResponseHeadersLength=64,RequestHeaderEncodingSelector=(name,message)=>System.Text.Encoding.Latin1,ResponseHeaderEncodingSelector=(name,message)=>System.Text.Encoding.Latin1};
                client=new HttpClient(handler,false){Timeout=System.Threading.Timeout.InfiniteTimeSpan};response=client.SendAsync(request,HttpCompletionOption.ResponseHeadersRead,lease.cancel.Token).GetAwaiter().GetResult();
                var raw=response.Content.ReadAsStreamAsync(lease.cancel.Token).GetAwaiter().GetResult();bodies.Add(raw);Stream body=raw;bool decompressed=gzip&&response.Content.Headers.ContentEncoding.Count==1&&response.Content.Headers.ContentEncoding.Contains("gzip");
                if(decompressed){body=new System.IO.Compression.GZipStream(body,System.IO.Compression.CompressionMode.Decompress,true);bodies.Add(body);}var wrapped=httpBodyWrapper(body)??throw new HostFault("null HTTP body wrapper");if(!bodies.Contains(wrapped))bodies.Add(wrapped);body=wrapped;
                var headerMap=new SortedDictionary<string,string>(StringComparer.Ordinal);int count=0;
                foreach(var header in response.Headers){string value=string.Join(", ",header.Value);count+=header.Key.Length+value.Length+4;headerMap[httpCanonical(header.Key)]=value;}
                foreach(var header in response.Content.Headers){if(decompressed&&(header.Key.Equals("Content-Encoding",StringComparison.OrdinalIgnoreCase)||header.Key.Equals("Content-Length",StringComparison.OrdinalIgnoreCase)))continue;string value=string.Join(", ",header.Value);count+=header.Key.Length+value.Length+4;headerMap[httpCanonical(header.Key)]=value;}
                if(count>65536)throw new Native.Reject("http: response headers exceed limit");var pairs=new List<string>();foreach(var p in headerMap){Native.binary(p.Value);pairs.Add(p.Key);pairs.Add(p.Value);}
                using var output=new MemoryStream();var buffer=new byte[8192];for(;;){int n=body.ReadAsync(buffer.AsMemory(),lease.cancel.Token).AsTask().GetAwaiter().GetResult();if(n==0)break;if(output.Length+n>limit)throw new Native.Reject("http: response body exceeds limit");output.Write(buffer,0,n);}result=new object[]{null,(long)response.StatusCode,pairs.ToArray(),output.ToArray()};
            }
            catch(Native.Reject e){rejection=e.Message;}catch(HttpRequestException e){rejection="http: transport failure";readFailure=e;}catch(IOException e){rejection="http: transport failure";readFailure=e;}catch(OperationCanceledException e){rejection="http: transport canceled";readFailure=e;}catch(Exception e){fault=new HostFault("HTTP adapter: "+e);}
            finally
            {
                Exception cleanup=null;for(int i=bodies.Count-1;i>=0;i--)try{bodies[i].Dispose();}catch(Exception e){if(!ReferenceEquals(e,readFailure))cleanup=e;}bodies.Clear();try{response?.Dispose();}catch(Exception e){if(!ReferenceEquals(e,readFailure))cleanup=e;}
                try{request?.Dispose();}catch(Exception e){cleanup=e;}request=null;try{client?.Dispose();}catch(Exception e){cleanup=e;}try{handler?.Dispose();}catch(Exception e){cleanup=e;}try{lease.Cleanup();}catch(Exception e){cleanup=e;}if(cleanup!=null)fault=new HostFault("HTTP cleanup: "+cleanup);
            }
            token.complete(result??new object[]{rejection,0L,Array.Empty<string>(),Array.Empty<byte>()},fault);
        }
    }
}
