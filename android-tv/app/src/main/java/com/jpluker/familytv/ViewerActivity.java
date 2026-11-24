package com.jpluker.familytv;

import android.app.Activity;
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

    @Override public void onCreate(Bundle saved) {
        super.onCreate(saved);
        server = getPreferences(MODE_PRIVATE).getString("server", "");
        if (server.isEmpty()) showPairing(); else openViewer("");
    }

    private void showPairing() {
        destroyViewer();
        ScrollView scroll = new ScrollView(this);
        LinearLayout form = new LinearLayout(this);
        form.setOrientation(LinearLayout.VERTICAL);
        int margin = (int)(32 * getResources().getDisplayMetrics().density);
        form.setPadding(margin, margin, margin, margin);
        scroll.addView(form);
        TextView heading = new TextView(this);
        heading.setText("Pair FamilyTV");
        heading.setTextSize(28);
        form.addView(heading);
        EditText address = field(form, "FamilyTV HTTPS address", InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI);
        address.setText(server);
        EditText code = field(form, "Pairing code from the parent dashboard", InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_FLAG_CAP_CHARACTERS);
        Button join = new Button(this);
        join.setText("Open viewer");
        form.addView(join);
        join.setOnClickListener(v -> {
            String origin = origin(address.getText().toString().trim());
            if (origin == null) { address.setError("Enter an HTTPS server address, including its port if needed."); address.requestFocus(); return; }
            String key = code.getText().toString().trim().replace("-", "").toUpperCase(Locale.ROOT);
            if (!key.matches("[A-Z2-7]{13}")) { code.setError("Enter the 13-character code shown on the parent dashboard."); code.requestFocus(); return; }
            server = origin;
            getPreferences(MODE_PRIVATE).edit().putString("server", server).apply();
            openViewer(key);
        });
        setContentView(scroll);
        (server.isEmpty() ? address : code).requestFocus();
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
    static String origin(String value) {
        try {
            URI uri = URI.create(value);
            if (!"https".equalsIgnoreCase(uri.getScheme()) || uri.getHost() == null || uri.getUserInfo() != null
                    || uri.getQuery() != null || uri.getFragment() != null || (!uri.getPath().isEmpty() && !uri.getPath().equals("/"))
                    || uri.getPort() == 0 || uri.getPort() > 65535) return null;
            return "https://" + uri.getRawAuthority();
        } catch (IllegalArgumentException ex) { return null; }
    }
    private boolean allowed(Uri url, boolean top) {
        if (!"https".equals(url.getScheme()) || !Uri.parse(server).getEncodedAuthority().equalsIgnoreCase(url.getEncodedAuthority())) return false;
        String path = url.getEncodedPath();
        if ("/watch".equals(path)) return true;
        return !top && path != null && !path.contains("%") && !path.contains("..") &&
                (path.startsWith("/assets/") || path.equals("/api/watch") || path.startsWith("/api/watch/"));
    }
    private void openViewer(String key) {
        destroyViewer();
        viewer = new WebView(this);
        viewer.getSettings().setJavaScriptEnabled(true);
        viewer.getSettings().setUserAgentString(viewer.getSettings().getUserAgentString() + " FamilyTV-AndroidTV");
        viewer.getSettings().setAllowFileAccess(false);
        viewer.getSettings().setAllowContentAccess(false);
        CookieManager.getInstance().setAcceptThirdPartyCookies(viewer, false);
        viewer.setWebViewClient(new WebViewClient() {
            @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest req) {
                if (req.isForMainFrame() && req.getUrl().toString().equals(server + "/watch/native")) {
                    startActivity(new Intent(ViewerActivity.this, PlaybackActivity.class).putExtra("server", server));
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
    @Override protected void onDestroy() {destroyViewer(); super.onDestroy();}
}
