package com.jpluker.familytv;

import android.app.Activity;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.os.SystemClock;
import android.view.WindowManager;
import android.webkit.CookieManager;
import android.widget.Toast;
import androidx.media3.common.C;
import androidx.media3.common.MediaItem;
import androidx.media3.common.MimeTypes;
import androidx.media3.common.PlaybackException;
import androidx.media3.common.Player;
import androidx.media3.datasource.okhttp.OkHttpDataSource;
import androidx.media3.exoplayer.ExoPlayer;
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory;
import androidx.media3.ui.PlayerView;
import java.io.IOException;
import java.util.Collections;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;
import org.json.JSONObject;

@androidx.annotation.OptIn(markerClass = androidx.media3.common.util.UnstableApi.class)
public final class PlaybackActivity extends Activity {
    private final Handler ui = new Handler(Looper.getMainLooper());
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final OkHttpClient http = new OkHttpClient.Builder().followRedirects(false)
            .followSslRedirects(false).build();
    private final OkHttpClient controlHttp = http.newBuilder().callTimeout(5, TimeUnit.SECONDS).build();
    private final Runnable deadline = this::finish;
    private String server, cookie, playingKey = "", mediaId = "";
    private boolean finishingVideo;
    private int generation, compatibilityAttempt;
    private long lastProgress;
    private MediaItem selectedItem;
    private ExoPlayer player;
    private PlayerView controls;

    @Override protected void onCreate(Bundle saved) {
        super.onCreate(saved);
        server = getIntent().getStringExtra("server");
        if (server == null || ViewerActivity.origin(server) == null) { finish(); return; }
        String cookies = CookieManager.getInstance().getCookie(server + "/api/watch");
        if (cookies != null) for (String part : cookies.split(";")) {
            if (part.trim().startsWith("familytv_viewer=")) cookie = part.trim();
        }
        if (cookie == null) { finish(); return; }
        controls = new PlayerView(this);
        setContentView(controls);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
    }
    @Override protected void onStart() {
        super.onStart();
        if (controls == null || isFinishing()) return;
        OkHttpDataSource.Factory source = new OkHttpDataSource.Factory(http);
        source.setDefaultRequestProperties(Collections.singletonMap("Cookie", cookie));
        player = new ExoPlayer.Builder(this).setMediaSourceFactory(new DefaultMediaSourceFactory(source)).build();
        controls.setPlayer(player);
        controls.requestFocus();
        player.addListener(new Player.Listener() {
            @Override public void onPlayerError(PlaybackException error) {
                if (selectedItem == null || compatibilityAttempt >= 2 || finishingVideo) { fail("This video could not be played."); return; }
                compatibilityAttempt++;
                String path = server + "/api/watch/media/" + android.net.Uri.encode(mediaId) + "/compatible"
                        + (compatibilityAttempt == 2 ? "?encode=1" : "");
                player.setMediaItem(selectedItem.buildUpon().setUri(path).build(), Math.max(0,player.getCurrentPosition()));
                player.prepare();
            }
            @Override public void onPlaybackStateChanged(int state) {
                if (state == Player.STATE_ENDED && !finishingVideo) finishVideo();
            }
        });
        playingKey = "";
        refresh(++generation);
    }
    private Request.Builder request(String path) {
        return new Request.Builder().url(server + path).header("Cookie", cookie);
    }
    private void refresh(int session) {
        if (session != generation || isFinishing() || finishingVideo) return;
        worker.execute(() -> {
            long sent = SystemClock.elapsedRealtime();
            try (Response response = controlHttp.newCall(request("/api/watch").build()).execute()) {
                if (!response.isSuccessful() || response.body() == null) throw new IOException();
                JSONObject state = new JSONObject(response.body().string());
                ui.post(() -> {
                    if (session != generation || isFinishing() || finishingVideo) return;
                    try {
                        showProgram(state, SystemClock.elapsedRealtime() - sent);
                        if (!isFinishing()) ui.postDelayed(() -> refresh(session), 1000);
                    } catch (Exception e) { fail("The viewing session is unavailable."); }
                });
            } catch (Exception e) {
                ui.post(() -> {if (session == generation) fail("The viewing session is unavailable.");});
            }
        });
    }
    private void showProgram(JSONObject state, long elapsed) throws Exception {
        JSONObject current = state.optJSONObject("current");
        if (current == null) { finish(); return; }
        long now = state.getLong("server_time") + elapsed;
        long end = state.optLong("deadline");
        ui.removeCallbacks(deadline);
        if (end > 0) {
            if (end <= now) { finish(); return; }
            ui.postDelayed(deadline, end - now);
        }
        String id = current.getString("id");
        String key = id + ":" + state.optBoolean("captions");
        if (key.equals(playingKey)) {
            if (player.isPlaying() && SystemClock.elapsedRealtime()-lastProgress >= 10000) saveProgress();
            return;
        }
        compatibilityAttempt = 0;
        String base = server + "/api/watch/media/" + android.net.Uri.encode(id);
        MediaItem.Builder item = new MediaItem.Builder().setUri(base + "/file");
        if (current.optBoolean("captions")) {
            MediaItem.SubtitleConfiguration track = new MediaItem.SubtitleConfiguration.Builder(android.net.Uri.parse(base + "/captions"))
                    .setMimeType(MimeTypes.TEXT_VTT).setLabel("Captions")
                    .setSelectionFlags(state.optBoolean("captions") ? C.SELECTION_FLAG_DEFAULT : 0).build();
            item.setSubtitleConfigurations(Collections.singletonList(track));
        }
        playingKey = key;
        mediaId = id;
        selectedItem=item.build();
        lastProgress = SystemClock.elapsedRealtime();
        player.setMediaItem(selectedItem, (long)(state.optDouble("resume",0)*1000));
        player.prepare();
        player.play();
    }
    private void saveProgress() {
        if (finishingVideo || player == null || mediaId.isEmpty() || player.getCurrentPosition()<=0) return;
        final double seconds = player.getCurrentPosition()/1000.0;
        final String id=mediaId;
        lastProgress=SystemClock.elapsedRealtime();
        worker.execute(() -> {
            try {
                String json=new JSONObject().put("id",id).put("kind","position").put("position",seconds).toString();
                Request req=request("/api/watch/bookmark").post(RequestBody.create(okhttp3.MediaType.parse("application/json"),json)).build();
                try (Response response=controlHttp.newCall(req).execute()) {
                    if (!response.isSuccessful()) return;
                }
            } catch(Exception ignored) { /* Playback continues. The next checkpoint retries. */ }
        });
    }
    private void finishVideo() {
        finishingVideo = true;
        final String id = mediaId;
        int session = generation;
        worker.execute(() -> {
            try {
                String json = new JSONObject().put("id", id).toString();
                Request req = request("/api/watch/finish").post(RequestBody.create(
                        okhttp3.MediaType.parse("application/json"), json)).build();
                try (Response response = controlHttp.newCall(req).execute()) {
                    if (!response.isSuccessful()) throw new IOException();
                }
                ui.post(() -> {if (session == generation) finish();});
            } catch (Exception e) { ui.post(() -> {if (session == generation) fail("Could not confirm the video ending.");}); }
        });
    }
    private void fail(String message) {Toast.makeText(this,message,Toast.LENGTH_SHORT).show(); finish();}
    @Override protected void onStop() {
        saveProgress();
        generation++;
        ui.removeCallbacksAndMessages(null);
        if (player != null) { controls.setPlayer(null); player.release(); player = null; }
        super.onStop();
    }
    @Override protected void onDestroy() {worker.shutdown(); super.onDestroy();}
}
