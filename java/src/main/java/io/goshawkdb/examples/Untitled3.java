// {{define "bcebbbd3-70d6-43cd-a4b6-c1609b88428f"}}
package io.goshawkdb.examples;

import java.io.StringReader;
import java.nio.ByteBuffer;

import io.goshawkdb.client.Certs;
import io.goshawkdb.client.Connection;
import io.goshawkdb.client.ConnectionFactory;
import io.goshawkdb.client.GoshawkObjRef;

public class Untitled3 {
    private static final String clusterCert = "...";
    private static final String clientKeyPair = "...";

    public static void main(final String[] args) throws Exception {
        final Certs certs = new Certs();
        certs.addClusterCertificate("myGoshawkDBCluster", clusterCert.getBytes());
        certs.parseClientPEM(new StringReader(clientKeyPair));
        final ConnectionFactory cf = new ConnectionFactory();
        try (final Connection conn = cf.connect(certs, "hostname")) {

            final String res = conn.runTransaction(txn -> {
                GoshawkObjRef root = txn.getRoots().get("myRoot1");
                root.set(ByteBuffer.wrap("Hello".getBytes()));
                return "success!";
            }).result;
            System.out.println(res);

            final String found = conn.runTransaction(txn -> {
                final ByteBuffer val = txn.getRoots().get("myRoot1").getValue();
                final byte[] ary = new byte[val.limit()];
                val.get(ary);
                return new String(ary);
            }).result;
            System.out.println("Found: " + found);

        } finally {
            cf.group.shutdownGracefully();
        }
    }
}
// {{end}}