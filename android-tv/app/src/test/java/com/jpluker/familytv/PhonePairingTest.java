package com.jpluker.familytv;

import com.google.zxing.qrcode.decoder.Decoder;
import com.google.zxing.qrcode.detector.Detector;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Locale;
import org.junit.Test;
import static org.junit.Assert.*;

public class PhonePairingTest {
    @Test public void scannedCodeContainsOnlyPublicChallenge() throws Exception {
        String server = "https://tv.example:8443";
        PhonePairing pairing = new PhonePairing(server);
        String scanned = new Decoder().decode(new Detector(pairing.matrix).detect().getBits()).getText();
        String challenge = scanned.substring(scanned.indexOf('=') + 1);
        assertTrue(scanned.startsWith(server + "/#pair-tv="));
        assertEquals(pairing.identifier, challenge.substring(0, 8));
        byte[] digest = MessageDigest.getInstance("SHA-256").digest(pairing.secret.getBytes(StandardCharsets.UTF_8));
        StringBuilder expected = new StringBuilder();
        for (byte b : digest) expected.append(String.format(Locale.ROOT, "%02x", b & 255));
        assertEquals(expected.toString(), challenge);
        assertFalse(scanned.contains(pairing.secret));
        assertNotEquals(pairing.secret, new PhonePairing(server).secret);
    }
}
