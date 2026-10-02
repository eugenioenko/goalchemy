package rt;

import java.io.ByteArrayOutputStream;
import java.util.ArrayList;
import java.util.Arrays;

/** Bounded DER framing only; key algorithms belong to maintained native providers. */
public final class Der {
    private Der() {}
    public record Node(int tag,byte[] content) {
        public byte[] encoded(){return value(tag,content);}
        public Node[] children() throws Native.Reject {return parseAll(content);}
    }
    public static Node one(byte[] bytes)throws Native.Reject {Node[] a=parseAll(bytes);if(a.length!=1)throw invalid();return a[0];}
    private static Native.Reject invalid(){return new Native.Reject("crypto: invalid input or key");}
    public static Node[] parseAll(byte[] bytes)throws Native.Reject {
        if(bytes.length>65536)throw invalid();var a=new ArrayList<Node>();int p=0;
        while(p<bytes.length) {
            if(a.size()>128||p+2>bytes.length)throw invalid();int tag=bytes[p++]&255,n=bytes[p++]&255;
            if((tag&31)==31)throw invalid();
            if(n>=128){int count=n&127;if(count==0||count>3||p+count>bytes.length||bytes[p]==0)throw invalid();n=0;for(int i=0;i<count;i++)n=(n<<8)|(bytes[p++]&255);if(n<128)throw invalid();}
            if(n>bytes.length-p)throw invalid();a.add(new Node(tag,Arrays.copyOfRange(bytes,p,p+n)));p+=n;
        }
        return a.toArray(Node[]::new);
    }
    public static byte[] value(int tag,byte[] content){var out=new ByteArrayOutputStream();out.write(tag);int n=content.length;if(n<128)out.write(n);else {int count=n<=255?1:n<=65535?2:3;out.write(128|count);for(int i=count-1;i>=0;i--)out.write(n>>>(8*i));}out.writeBytes(content);return out.toByteArray();}
    public static byte[] join(byte[]... values){var out=new ByteArrayOutputStream();for(byte[] v:values)out.writeBytes(v);return out.toByteArray();}
    public static byte[] seq(byte[]... values){return value(48,join(values));}
    public static byte[] integer(java.math.BigInteger n){return value(2,n.toByteArray());}
    public static final byte[] RSA_ALGORITHM=java.util.HexFormat.of().parseHex("300d06092a864886f70d0101010500");
    public static final byte[] EC_ALGORITHM=java.util.HexFormat.of().parseHex("301306072a8648ce3d020106082a8648ce3d030107");
    public static byte[] rawSignature(byte[] der)throws Native.Reject {
        if(der.length>72)throw invalid();Node root=one(der);if(root.tag!=48)throw invalid();Node[] parts=root.children();if(parts.length!=2)throw invalid();byte[] result=new byte[64];
        for(int j=0;j<2;j++){Node part=parts[j];byte[] b=part.content;if(part.tag!=2||b.length==0||b.length>33||(b[0]&128)!=0||b.length>1&&b[0]==0&&(b[1]&128)==0)throw invalid();int start=b.length==33?1:0;if(start==1&&b[0]!=0)throw invalid();System.arraycopy(b,start,result,(j+1)*32-(b.length-start),b.length-start);}
        return result;
    }
    public static byte[] derSignature(byte[] raw)throws Native.Reject {if(raw.length!=64)throw invalid();return seq(integer(new java.math.BigInteger(1,Arrays.copyOfRange(raw,0,32))),integer(new java.math.BigInteger(1,Arrays.copyOfRange(raw,32,64))));}
    public static byte[] privateWithPoint(byte[] pkcs8,byte[] point)throws Native.Reject {
        Node root=one(pkcs8);Node[] outer=root.children();if(root.tag!=48||outer.length<3||outer[2].tag!=4)throw invalid();
        Node inner=one(outer[2].content);if(inner.tag!=48)throw invalid();Node[] fields=inner.children();var content=new ByteArrayOutputStream();for(Node n:fields)if(n.tag!=161)content.writeBytes(n.encoded());
        content.writeBytes(value(161,value(3,join(new byte[]{0},point))));outer[2]=new Node(4,value(48,content.toByteArray()));var result=new ByteArrayOutputStream();for(Node n:outer)result.writeBytes(n.encoded());return value(48,result.toByteArray());
    }
    public static byte[] includedPoint(byte[] pkcs8)throws Native.Reject {
        Node root=one(pkcs8);Node[] outer=root.children();if(root.tag!=48||outer.length<3||outer[2].tag!=4)throw invalid();Node inner=one(outer[2].content);if(inner.tag!=48)throw invalid();byte[] point=null;
        for(Node n:inner.children())if(n.tag==161){if(point!=null)throw invalid();Node bits=one(n.content);if(bits.tag!=3||bits.content.length!=66||bits.content[0]!=0||bits.content[1]!=4)throw invalid();point=Arrays.copyOfRange(bits.content,1,66);}
        return point;
    }
}
