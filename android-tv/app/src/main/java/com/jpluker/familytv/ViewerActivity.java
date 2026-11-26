package com.jpluker.familytv;

import android.app.Activity;
import android.app.AlertDialog;
import android.graphics.Bitmap;
import android.widget.ImageView;
import com.google.zxing.WriterException;
import com.google.zxing.common.BitMatrix;
import java.security.NoSuchAlgorithmException;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.text.InputType;
import android.view.View;
import android.webkit.CookieManager;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.TextView;
import java.io.ByteArrayInputStream;
import java.net.URI;
import java.util.Locale;

public final class ViewerActivity extends Activity {
    private WebView viewer;
    private String server;
    private ServerDiscovery discovery;
    private int pairingGeneration;

    @Override public void onCreate(Bundle saved) {
        super.onCreate(saved);
        ViewerWindow.configure(this);
        server = getPreferences(MODE_PRIVATE).getString("server", "");
        if (origin(server, getString(R.string.local_test_origin)) == null) showPairing(); else openViewer("");
    }

    private void showPairing() {
        destroyViewer();
        ScrollView scroll = new ScrollView(this);
        LinearLayout form = new LinearLayout(this);
        form.setOrientation(LinearLayout.VERTICAL);
        boolean tv = getResources().getBoolean(R.bool.television);
        int margin = (int)((tv ? 32 : 20) * getResources().getDisplayMetrics().density);
        form.setPadding(margin, margin, margin, margin);
        scroll.addView(form);
        TextView heading = new TextView(this);
        heading.setText("Pair FamilyTV");
        heading.setTextSize(36);
        heading.setTypeface(getResources().getFont(R.font.jua));
        heading.setPadding(0, 0, 0, margin / 2);
        form.addView(heading);
        int generation = ++pairingGeneration;
        stopDiscovery();
        TextView found = new TextView(this);
        found.setText("Searching for FamilyTV servers…");
        form.addView(found);
        LinearLayout choices=new LinearLayout(this);
        choices.setOrientation(LinearLayout.VERTICAL);
        form.addView(choices);
        Button advanced=new Button(this);
        advanced.setText("Advanced server address");
        form.addView(advanced);
        LinearLayout manual=new LinearLayout(this);
        manual.setOrientation(LinearLayout.VERTICAL);
        manual.setVisibility(View.GONE);
        form.addView(manual);
        EditText address = field(manual, getString(R.string.local_test_origin).isEmpty() ? "FamilyTV HTTPS address" : "FamilyTV server address", InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        address.setText(server);
        advanced.setOnClickListener(v -> {manual.setVisibility(View.VISIBLE);address.requestFocus();});
        discovery=new ServerDiscovery(this, candidate -> {
            if(generation!=pairingGeneration || isDestroyed())return;
            if(choices.getChildCount()==0 && address.getText().length()==0)address.setText(candidate);
            found.setText("FamilyTV server: "+address.getText());
            Button choice=new Button(this);
            choice.setText(candidate);choices.addView(choice);
            choice.setOnClickListener(v->{address.setText(candidate);found.setText("FamilyTV server: "+candidate);});
        });
        discovery.start(getString(R.string.local_test_origin));
        found.postDelayed(()->{if(generation==pairingGeneration && choices.getChildCount()==0)found.setText("No server found. Use Advanced server address.");},4500);
        Button parent=new Button(this);
        parent.setText("Parent mode");
        form.addView(parent);
        parent.setOnClickListener(v->{
            String target=origin(address.getText().toString().trim(),getString(R.string.local_test_origin));
            if(target==null){manual.setVisibility(View.VISIBLE);address.setError("Choose or enter your FamilyTV server.");address.requestFocus();return;}
            try{startActivity(new Intent(Intent.ACTION_VIEW,Uri.parse(target+"/?parent-mode=1")));}
            catch(android.content.ActivityNotFoundException e){found.setText("Open "+target+"/?parent-mode=1 in your parent browser.");}
        });
        EditText code = field(form, "Pairing code from the parent dashboard", InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_FLAG_CAP_CHARACTERS);
        Button join = new Button(this);
        join.setText("Open viewer");
        addPairingButton(form, join, margin / 3);
        join.setOnClickListener(v -> {
            String origin = origin(address.getText().toString().trim(), getString(R.string.local_test_origin));
            if (origin == null) { manual.setVisibility(View.VISIBLE); address.setError("Enter an HTTPS address or the configured local test address."); address.requestFocus(); return; }
            String key = code.getText().toString().trim().replace("-", "").toUpperCase(Locale.ROOT);
            if (!key.matches("[A-Z2-7]{6}|[A-Z2-7]{13}")) { code.setError("Enter the six-character code shown on the parent dashboard."); code.requestFocus(); return; }
            server = origin;
            getPreferences(MODE_PRIVATE).edit().putString("server", server).apply();
            openViewer(key);
        });
        if (tv) addPhonePairing(form, address, margin);
        setContentView(scroll);
        if (tv) code.requestFocus();
    }

    private void addPhonePairing(LinearLayout form, EditText address, int margin) {
        Button scan = new Button(this);
        scan.setText("Pair using a phone");
        addPairingButton(form, scan, margin / 3);
        scan.setOnClickListener(v -> {
            String origin = origin(address.getText().toString().trim(), getString(R.string.local_test_origin));
            if (origin == null) { address.setError("Check your FamilyTV server address."); address.requestFocus(); return; }
            showPhonePairing(origin);
        });
    }
    private void addPairingButton(LinearLayout form, Button button, int spacing) {
        LinearLayout.LayoutParams layout = new LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT);
        layout.topMargin = spacing;
        form.addView(button, layout);
    }

    private void showPhonePairing(String origin) {
        try {
            PhonePairing pairing = new PhonePairing(origin);
            BitMatrix matrix = pairing.matrix;
            Bitmap bitmap = Bitmap.createBitmap(matrix.getWidth(), matrix.getHeight(), Bitmap.Config.ARGB_8888);
            int[] row = new int[matrix.getWidth()];
            for (int y = 0; y < matrix.getHeight(); y++) {
                for (int x = 0; x < row.length; x++) row[x] = matrix.get(x, y) ? 0xff000000 : 0xffffffff;
                bitmap.setPixels(row, 0, row.length, 0, y, row.length, 1);
            }
            ImageView image = new ImageView(this);
            image.setAdjustViewBounds(true);
            image.setImageBitmap(bitmap);
            image.setContentDescription("Scan with the parent's phone to choose a profile");
            new AlertDialog.Builder(this).setTitle("Pair TV " + pairing.identifier)
                    .setMessage("Scan with your phone. Confirm this TV code under Household devices, then open the viewer.")
                    .setView(image).setNegativeButton("Cancel", null)
                    .setPositiveButton("Open paired viewer", (dialog, which) -> {
                        server = origin;
                        getPreferences(MODE_PRIVATE).edit().putString("server", server).apply();
                        openViewer(pairing.secret);
                    }).show();
        } catch (NoSuchAlgorithmException | WriterException | IllegalArgumentException ex) {
            new AlertDialog.Builder(this).setMessage("QR pairing is unavailable. Use the pairing code instead.")
                    .setPositiveButton("OK", null).show();
        }
    }
    private EditText field(LinearLayout form, String label, int type) {
        TextView text = new TextView(this);
        text.setText(label);
        form.addView(text);
        EditText input = new EditText(this);
        input.setSingleLine(true);
        input.setInputType(type);
        input.setId(View.generateViewId());
        text.setLabelFor(input.getId());
        form.addView(input);
        return input;
    }
    static String origin(String value, String localTestOrigin) {
        try {
            URI uri = URI.create(value);
            if (uri.getHost() == null || uri.getUserInfo() != null
                    || uri.getQuery() != null || uri.getFragment() != null || (!uri.getPath().isEmpty() && !uri.getPath().equals("/"))
                    || uri.getPort() == 0 || uri.getPort() > 65535) return null;
            if ("https".equalsIgnoreCase(uri.getScheme())) return "https://" + uri.getRawAuthority();
            String origin = uri.getScheme() + "://" + uri.getRawAuthority();
            return origin.equals(localTestOrigin) ? origin : null;
        } catch (IllegalArgumentException ex) { return null; }
    }
    private boolean allowed(Uri url, boolean top) {
        Uri configured = Uri.parse(server);
        if (!configured.getScheme().equals(url.getScheme()) || !configured.getEncodedAuthority().equalsIgnoreCase(url.getEncodedAuthority())) return false;
        String path = url.getEncodedPath();
        if ("/watch".equals(path)) return true;
        return !top && path != null && !path.contains("%") && !path.contains("..") &&
                (path.startsWith("/assets/") || path.equals("/api/watch") || path.startsWith("/api/watch/"));
    }
    private void stopDiscovery() {
        if(discovery!=null){discovery.close();discovery=null;}
    }
    private void openViewer(String key) {
        pairingGeneration++;stopDiscovery();
        destroyViewer();
        viewer = new WebView(this);
        viewer.getSettings().setJavaScriptEnabled(true);
        viewer.getSettings().setUserAgentString(viewer.getSettings().getUserAgentString() +
                (getResources().getBoolean(R.bool.television) ? " FamilyTV-AndroidTV" : " FamilyTV-AndroidTV FamilyTV-Handheld"));
        viewer.getSettings().setAllowFileAccess(false);
        viewer.getSettings().setAllowContentAccess(false);
        CookieManager.getInstance().setAcceptThirdPartyCookies(viewer, false);
        viewer.setWebViewClient(new WebViewClient() {
            @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest req) {
                if (req.isForMainFrame() && req.getUrl().toString().equals(server + "/watch/native")) {
                    startActivityForResult(new Intent(ViewerActivity.this, PlaybackActivity.class).putExtra("server", server), 1);
                    return true;
                }
                return !allowed(req.getUrl(), req.isForMainFrame());
            }
            @Override public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest req) {
                if (allowed(req.getUrl(), req.isForMainFrame())) return null;
                return new WebResourceResponse("text/plain","UTF-8",new ByteArrayInputStream(new byte[0]));
            }
        });
        setContentView(viewer);
        viewer.requestFocus();
        viewer.loadUrl(server + "/watch" + (key.isEmpty() ? "" : "#" + key));
    }
    private void destroyViewer() {
        if (viewer == null) return;
        setContentView(new View(this));
        viewer.stopLoading();
        viewer.destroy();
        viewer = null;
    }
    @Override protected void onActivityResult(int request,int result,Intent data) {
        super.onActivityResult(request,result,data);
        if(request==1 && result==RESULT_OK && viewer!=null)
            viewer.evaluateJavascript("window.dispatchEvent(new Event('familytv-player-back'))",null);
    }
    @Override public void onBackPressed() {
        if (viewer == null) { super.onBackPressed(); return; }
        viewer.evaluateJavascript("window.dispatchEvent(new Event('familytv-back',{cancelable:true}))", result -> {
            if ("true".equals(result)) showPairing();
        });
    }
    @Override protected void onPause() {
        if (viewer != null) {viewer.evaluateJavascript("document.querySelector('video')?.pause()", null); viewer.onPause();}
        super.onPause();
    }
    @Override protected void onResume() {super.onResume(); if (viewer != null) viewer.onResume();}
    @Override protected void onDestroy() {pairingGeneration++;stopDiscovery();destroyViewer(); super.onDestroy();}
}
