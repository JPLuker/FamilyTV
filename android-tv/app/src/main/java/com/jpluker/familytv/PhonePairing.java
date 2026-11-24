package com.jpluker.familytv;

import com.google.zxing.BarcodeFormat;
import com.google.zxing.WriterException;
import com.google.zxing.common.BitMatrix;
import com.google.zxing.qrcode.QRCodeWriter;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.UUID;
import java.util.Locale;

final class PhonePairing {
    final String secret;
    final String identifier;
    final BitMatrix matrix;

    PhonePairing(String origin) throws WriterException, NoSuchAlgorithmException {
        secret = UUID.randomUUID().toString();
        byte[] digest = MessageDigest.getInstance("SHA-256").digest(secret.getBytes(StandardCharsets.UTF_8));
        StringBuilder challenge = new StringBuilder();
        for (byte b : digest) challenge.append(String.format(Locale.ROOT, "%02x", b & 255));
        identifier = challenge.substring(0, 8);
        matrix = new QRCodeWriter().encode(origin + "/#pair-tv=" + challenge, BarcodeFormat.QR_CODE, 384, 384);
    }
}
