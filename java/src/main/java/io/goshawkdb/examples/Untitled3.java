// {{define "bcebbbd3-70d6-43cd-a4b6-c1609b88428f"}}
package io.goshawkdb.examples;

import java.io.StringReader;
import java.nio.ByteBuffer;

import io.goshawkdb.client.Certs;
import io.goshawkdb.client.Connection;
import io.goshawkdb.client.ConnectionFactory;
import io.goshawkdb.client.RefCap;
import io.goshawkdb.client.TransactionResult;
import io.goshawkdb.client.ValueRefs;

public class Untitled3 {
    private static final String clusterCert = "...";
    private static final String clientKeyPair = "...";

    public static void main(final String[] args) throws Exception {
        Certs certs = new Certs();
        certs.addClusterCertificate("myGoshawkDBCluster", clusterCert.getBytes());
        certs.parseClientPEM(new StringReader(clientKeyPair));
        try (ConnectionFactory cf = new ConnectionFactory()) {
            try (Connection conn = cf.connect(certs, "hostname")) {

                TransactionResult<String> outcome = conn.transact(txn -> {
                    RefCap rootRef = txn.root("myRoot1");
                    if (rootRef == null) {
                        throw new RuntimeException("No root 'myRoot1' found");
                    }
                    txn.write(rootRef, ByteBuffer.wrap("Hello".getBytes()));
                    return "success!";
                });
                System.out.println("" + outcome.result + ", " + outcome.cause);
                outcome.getResultOrRethrow();

                outcome = conn.transact(txn -> {
                    RefCap rootRef = txn.root("myRoot1");
                    if (rootRef == null) {
                        throw new RuntimeException("No root 'myRoot1' found");
                    }
                    ValueRefs rootValueRefs = txn.read(rootRef);
                    if (txn.restartNeeded()) {
                        return null;
                    }
                    byte[] ary = new byte[rootValueRefs.value.limit()];
                    rootValueRefs.value.get(ary);
                    return new String(ary);
                });
                System.out.println("Found: " + outcome.result + ", " + outcome.cause);
                outcome.getResultOrRethrow();
            }
        }
    }
}
// {{end}}