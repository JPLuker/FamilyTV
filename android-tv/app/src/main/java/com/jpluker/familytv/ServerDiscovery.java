package com.jpluker.familytv;

import android.content.Context;
import android.net.nsd.NsdManager;
import android.net.nsd.NsdServiceInfo;
import android.os.Handler;
import android.os.Looper;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.ArrayDeque;
import java.util.HashSet;
import java.util.Set;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.function.Consumer;
import org.json.JSONObject;

/** Browse advertised instances and verify the public health endpoint before displaying one. */
final class ServerDiscovery implements NsdManager.DiscoveryListener {
    private final NsdManager manager;
    private final Handler main = new Handler(Looper.getMainLooper());
    private final ExecutorService worker = Executors.newSingleThreadExecutor();
    private final Set<String> candidates = new HashSet<>();
    private final ArrayDeque<NsdServiceInfo> pending = new ArrayDeque<>();
    private final Consumer<String> found;
    private String local;
    private boolean browsing, resolving, closed;

    ServerDiscovery(Context context, Consumer<String> found) {
        manager=context.getSystemService(NsdManager.class);
        this.found=found;
    }
    void start(String localOrigin) {
        local=localOrigin;
        if(!local.isEmpty()) verify(local);
        try {
            manager.discoverServices("_familytv._tcp.",NsdManager.PROTOCOL_DNS_SD,this);
            browsing=true;
        } catch(RuntimeException ignored) { /* Manual entry remains available. */ }
    }
    public void onServiceFound(NsdServiceInfo service) {
        main.post(()->{if(!closed && pending.size()<12){pending.add(service);resolveNext();}});
    }
    private void resolveNext() {
        if(closed || resolving || pending.isEmpty())return;
        resolving=true;
        try {
            manager.resolveService(pending.remove(),new NsdManager.ResolveListener(){
                public void onResolveFailed(NsdServiceInfo info,int error){main.post(()->{resolving=false;resolveNext();});}
                public void onServiceResolved(NsdServiceInfo info){
                    main.post(()->{
                        resolving=false;
                        byte[] value=info.getAttributes().get("origin");
                        if(!closed && value!=null)verify(new String(value,StandardCharsets.UTF_8));
                        resolveNext();
                    });
                }
            });
        } catch(RuntimeException e){resolving=false;}
    }
    private void verify(String value) {
        String origin=ViewerActivity.origin(value,local);
        if(closed || origin==null || candidates.size()>=12 || !candidates.add(origin))return;
        worker.execute(()->{
            HttpURLConnection conn=null;
            try {
                conn=(HttpURLConnection)new URL(origin+"/api/health").openConnection();
                conn.setInstanceFollowRedirects(false);
                conn.setConnectTimeout(1400);conn.setReadTimeout(1400);
                if(conn.getResponseCode()!=200)return;
                try(InputStream in=conn.getInputStream()){
                    java.io.ByteArrayOutputStream body=new java.io.ByteArrayOutputStream();
                    byte[] buffer=new byte[256];int count;
                    while((count=in.read(buffer))!=-1){if(body.size()+count>1024)return;body.write(buffer,0,count);}
                    if("FamilyTV".equals(new JSONObject(body.toString("UTF-8")).optString("service")))
                        main.post(()->{if(!closed)found.accept(origin);});
                }
            }catch(Exception ignored){ /* Stale advertisements and offline hosts are omitted. */ }
            finally{if(conn!=null)conn.disconnect();}
        });
    }
    void close(){
        closed=true;worker.shutdownNow();
        if(browsing)try{manager.stopServiceDiscovery(this);}catch(RuntimeException ignored){}
    }
    public void onDiscoveryStarted(String type){}
    public void onDiscoveryStopped(String type){}
    public void onServiceLost(NsdServiceInfo service){}
    public void onStartDiscoveryFailed(String type,int error){}
    public void onStopDiscoveryFailed(String type,int error){}
}
