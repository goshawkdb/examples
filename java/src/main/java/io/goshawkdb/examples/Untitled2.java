// {{define "5adf1eb2-423d-428e-882a-7b1d2093acba"}}
package io.goshawkdb.examples;

import java.io.StringReader;

import io.goshawkdb.client.Certs;
import io.goshawkdb.client.Connection;
import io.goshawkdb.client.ConnectionFactory;

public class Untitled2 {
    private static final String clusterCert = "...";
    private static final String clientKeyPair = "...";

    public static void main(final String[] args) throws Exception {
        Certs certs = new Certs();
        certs.addClusterCertificate("myGoshawkDBCluster", clusterCert.getBytes());
        certs.parseClientPEM(new StringReader(clientKeyPair));
        try (ConnectionFactory cf = new ConnectionFactory()) {
            try (Connection conn = cf.connect(certs, "hostname")) {
                String res = conn.runTransaction(txn ->
                        "Hello World"
                ).result;
                System.out.println(res);
            }
        }
    }
}
// {{end}}