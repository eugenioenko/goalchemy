package rt;

import java.io.IOException;
import java.nio.charset.StandardCharsets;

/** Byte-exact output to standard error and standard output. */
public final class Out {
    private Out() {}

    public static void stderr(String s) {
        try {
            System.err.write(s.getBytes(StandardCharsets.ISO_8859_1));
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
        System.err.flush();
    }

    public static void stdout(String s) {
        try {
            System.out.write(s.getBytes(StandardCharsets.ISO_8859_1));
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
        System.out.flush();
    }
}
