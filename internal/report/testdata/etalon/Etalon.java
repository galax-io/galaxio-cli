// Etalon prints the percentiles the real t-digest libraries give for one run:
// the reference galaxio report's percentiles are held to (constitution
// Principle II; specs/005-report-summary/research.md §18).
//
//   java Etalon.java t-digest-3.1.jar t-digest-3.3.jar < samples.tsv > etalon.tsv
//
// Standard input holds one request a line, in log order: "ok" or "failed", a
// tab, and its response time in whole milliseconds. The output names how many
// lines it read and their SHA-256, so that an etalon.tsv can be held to the log
// it was made from without running this again.
//
// Gatling 3.11 and 3.12 feed every request, in log order, to an
// AVLTreeDigest(100) of t-digest 3.1 for all requests and to another for its
// outcome, and print Math.round(quantile(rank / 100)). That digest draws on an
// unseeded java.util.Random, so Gatling can print two values for one log; here
// it is built once for each seed from 1 to 200 and every value it gave is
// printed. MergingDigest(100) of t-digest 3.3, which draws on no generator, is
// built once beside it. Each jar is loaded by a class loader of its own, so the
// two versions of one library never meet.
import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.lang.reflect.Modifier;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Paths;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.HexFormat;
import java.util.List;
import java.util.Random;
import java.util.StringJoiner;
import java.util.TreeSet;

public class Etalon {
    static final String[] COLUMNS = {"all", "ok", "failed"};
    static final int[] RANKS = {50, 75, 95, 99};
    static final int SEEDS = 200;
    static final double COMPRESSION = 100;

    public static void main(String[] args) throws Exception {
        if (args.length != 2) {
            System.err.println("usage: java Etalon.java <t-digest-3.1.jar> <t-digest-3.3.jar> < samples.tsv");
            System.exit(2);
        }

        List<List<Double>> columns = new ArrayList<>();
        for (int c = 0; c < COLUMNS.length; c++) {
            columns.add(new ArrayList<>());
        }

        byte[] input = System.in.readAllBytes();
        String sha256 = HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(input));
        String text = new String(input, StandardCharsets.UTF_8);
        int samples = 0;

        for (String line : text.isEmpty() ? new String[0] : text.split("\n")) {
            samples++;

            String[] fields = line.split("\t", -1);
            if (fields.length != 2) {
                throw new IllegalArgumentException("not an outcome and a response time: " + line);
            }

            double ms = Long.parseLong(fields[1]);
            columns.get(0).add(ms);

            if (fields[0].equals("ok")) {
                columns.get(1).add(ms);
            } else if (fields[0].equals("failed")) {
                columns.get(2).add(ms);
            } else {
                throw new IllegalArgumentException("not ok or failed: " + line);
            }
        }

        Digest avl = new Digest(args[0], "com.tdunning.math.stats.AVLTreeDigest");
        Digest merging = new Digest(args[1], "com.tdunning.math.stats.MergingDigest");

        StringBuilder out = new StringBuilder();
        out.append("# t-digest 3.1 AVLTreeDigest(100) over seeds 1-").append(SEEDS)
            .append(" and t-digest 3.3 MergingDigest(100), each read as Math.round(quantile(rank / 100))\n");
        out.append("# samples ").append(samples).append(" sha256 ").append(sha256).append('\n');
        out.append("column\trank\tavl-3.1\tmerging-3.3\n");

        for (int c = 0; c < COLUMNS.length; c++) {
            List<Double> values = columns.get(c);

            List<TreeSet<Long>> avlValues = new ArrayList<>();
            for (int r = 0; r < RANKS.length; r++) {
                avlValues.add(new TreeSet<>());
            }

            long[] mergingValues = new long[RANKS.length];

            if (!values.isEmpty()) {
                for (int seed = 1; seed <= SEEDS; seed++) {
                    Object digest = avl.build(values, new Random(seed));
                    for (int r = 0; r < RANKS.length; r++) {
                        avlValues.get(r).add(avl.percentile(digest, RANKS[r]));
                    }
                }

                Object digest = merging.build(values, null);
                for (int r = 0; r < RANKS.length; r++) {
                    mergingValues[r] = merging.percentile(digest, RANKS[r]);
                }
            }

            for (int r = 0; r < RANKS.length; r++) {
                out.append(COLUMNS[c]).append('\t').append(RANKS[r]).append('\t');

                if (values.isEmpty()) {
                    out.append("-\t-\n");
                    continue;
                }

                StringJoiner joined = new StringJoiner(",");
                for (long v : avlValues.get(r)) {
                    joined.add(Long.toString(v));
                }

                out.append(joined).append('\t').append(mergingValues[r]).append('\n');
            }
        }

        System.out.print(out);
    }

    // Digest is one digest class of one jar, reached by reflection so that the
    // two versions of the library load side by side.
    static final class Digest {
        final Constructor<?> constructor;
        final Method add;
        final Method quantile;
        final Field generator;

        Digest(String jar, String className) throws Exception {
            ClassLoader loader = new URLClassLoader(new URL[] {Paths.get(jar).toUri().toURL()}, ClassLoader.getPlatformClassLoader());
            Class<?> digest = Class.forName(className, true, loader);

            constructor = digest.getConstructor(double.class);
            add = digest.getMethod("add", double.class);
            quantile = digest.getMethod("quantile", double.class);

            Field found = null;
            for (Class<?> k = digest; k != null && found == null; k = k.getSuperclass()) {
                for (Field f : k.getDeclaredFields()) {
                    if (f.getType() == Random.class && !Modifier.isStatic(f.getModifiers())) {
                        found = f;
                    }
                }
            }

            if (found != null) {
                found.setAccessible(true);
            }

            generator = found;
        }

        Object build(List<Double> values, Random random) throws Exception {
            Object digest = constructor.newInstance(COMPRESSION);

            if (random != null) {
                if (generator == null) {
                    throw new IllegalStateException(constructor.getDeclaringClass() + " draws on no generator to seed");
                }

                generator.set(digest, random);
            }

            for (double v : values) {
                add.invoke(digest, v);
            }

            return digest;
        }

        long percentile(Object digest, int rank) throws Exception {
            return Math.round((double) quantile.invoke(digest, rank / 100.0));
        }
    }
}
