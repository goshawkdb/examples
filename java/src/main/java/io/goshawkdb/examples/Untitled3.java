// {{define "bcebbbd3-70d6-43cd-a4b6-c1609b88428f"}}
package io.goshawkdb.examples;

import java.io.StringReader;
import java.nio.ByteBuffer;

import io.goshawkdb.client.Certs;
import io.goshawkdb.client.Connection;
import io.goshawkdb.client.ConnectionFactory;
import io.goshawkdb.client.GoshawkObjRef;
import io.goshawkdb.client.TransactionResult;

public class Untitled3 {
    private static final String clusterCert = "...";
    private static final String clientKeyPair = "...";

    public static void main(final String[] args) throws Exception {
        Certs certs = new Certs();
        certs.addClusterCertificate("myGoshawkDBCluster", clusterCert.getBytes());
        certs.parseClientPEM(new StringReader(clientKeyPair));
        try (ConnectionFactory cf = new ConnectionFactory()) {
            try (Connection conn = cf.connect(certs, "hostname")) {

                TransactionResult<String> outcome = conn.runTransaction(txn -> {
                    GoshawkObjRef root = txn.getRoots().get("myRoot1");
                    if (root == null) {
                        throw new RuntimeException("No root 'myRoot1' found");
                    }
                    root.set(ByteBuffer.wrap("Hello".getBytes()));
                    return "success!";
                });
                System.out.println("" + outcome.result + ", " + outcome.cause);

                outcome = conn.runTransaction(txn -> {
                    GoshawkObjRef root = txn.getRoots().get("myRoot1");
                    if (root == null) {
                        throw new RuntimeException("No root 'myRoot1' found");
                    }
                    ByteBuffer val = root.getValue();
                    byte[] ary = new byte[val.limit()];
                    val.get(ary);
                    return new String(ary);
                });
                System.out.println("Found: " + outcome.result + ", " + outcome.cause);
            }
        }
    }
}
// {{end}}