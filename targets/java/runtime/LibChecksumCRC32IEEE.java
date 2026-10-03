package rt;

import java.util.zip.CRC32;

/** Native IEEE CRC-32 over the logical byte slice, without copying its storage. */
public final class LibChecksumCRC32IEEE {
    private LibChecksumCRC32IEEE() { }

    public static long libChecksumCRC32IEEE(Slice data) {
        if (data.l == 0) return 0;
        CRC32 checksum = new CRC32();
        checksum.update((byte[]) data.a, data.o, data.l);
        return checksum.getValue();
    }
}
