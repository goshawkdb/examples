// {{define "bb832346-d5fe-41dd-ac93-1cc3e4aea683"}}
package io.goshawkdb.examples;

import java.io.StringReader;
import java.nio.ByteBuffer;

import io.goshawkdb.client.Certs;
import io.goshawkdb.client.Connection;
import io.goshawkdb.client.ConnectionFactory;
import io.goshawkdb.client.GoshawkObjRef;

public class Untitled4 {
    private static final String clusterCert = "...";
    private static final String clientKeyPair = "...";

    public static void main(final String[] args) throws Exception {
        final Certs certs = new Certs();
        certs.addClusterCertificate("myGoshawkDBCluster", clusterCert.getBytes());
        certs.parseClientPEM(new StringReader(clientKeyPair));
        final ConnectionFactory cf = new ConnectionFactory();
        try (final Connection conn = cf.connect(certs, "hostname")) {

            final String res = conn.runTransaction(txn -> {
                final GoshawkObjRef obj = txn.createObject(ByteBuffer.wrap("a new value for a new object".getBytes()));
                txn.getRoots().get("myRoot1").set(null, obj); // root now has a single reference to our new object
                return "success!";
            }).result;
            System.out.println(res);

            final String found = conn.runTransaction(txn -> {
                final GoshawkObjRef[] objs = txn.getRoots().get("myRoot1").getReferences();
                final ByteBuffer val = objs[0].getValue();
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