// {{define "bb832346-d5fe-41dd-ac93-1cc3e4aea683"}}
package io.goshawkdb.examples;

import java.io.StringReader;
import java.nio.ByteBuffer;

import io.goshawkdb.client.Certs;
import io.goshawkdb.client.Connection;
import io.goshawkdb.client.ConnectionFactory;
import io.goshawkdb.client.GoshawkObjRef;
import io.goshawkdb.client.TransactionResult;

public class Untitled4 {
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
                    GoshawkObjRef obj = txn.createObject(ByteBuffer.wrap("a new value for a new object".getBytes()));
                    root.set(null, obj); // root now has a single reference to our new object
                    return "success!";
                });
                System.out.println("" + outcome.result + ", " + outcome.cause);
                outcome.getResultOrRethrow();

                outcome = conn.runTransaction(txn -> {
                    GoshawkObjRef root = txn.getRoots().get("myRoot1");
                    if (root == null) {
                        throw new RuntimeException("No root 'myRoot1' found");
                    }
                    GoshawkObjRef[] objs = root.getReferences();
                    ByteBuffer val = objs[0].getValue();
                    byte[] ary = new byte[val.limit()];
                    val.get(ary);
                    return new String(ary);
                });
                System.out.println("Found: " + outcome.result + ", " + outcome.cause);
                outcome.getResultOrRethrow();

            }
        }
    }
}
// {{end}}