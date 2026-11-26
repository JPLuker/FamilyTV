package com.jpluker.familytv;

import android.content.Intent;
import android.content.res.Resources;
import android.content.res.XmlResourceParser;
import org.robolectric.RuntimeEnvironment;
import org.xmlpull.v1.XmlPullParser;
import android.content.pm.ActivityInfo;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Button;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.robolectric.Robolectric;
import org.robolectric.RobolectricTestRunner;
import org.robolectric.annotation.Config;
import static org.junit.Assert.*;

@RunWith(RobolectricTestRunner.class)
@Config(sdk = 34)
public class DeviceConfigurationTest {
    @Test @Config(qualifiers = "port")
    public void phoneHasLauncherAndManualPairing() { checkConfiguration(false); }

    @Test @Config(qualifiers = "sw600dp-land")
    public void tabletKeepsTouchLayoutInLandscape() { checkConfiguration(false); }

    @Test @Config(qualifiers = "land-television")
    public void televisionKeepsRemoteAndPhonePairing() { checkConfiguration(true); }

    private void checkConfiguration(boolean tv) {
        ViewerActivity activity = Robolectric.buildActivity(ViewerActivity.class).setup().get();
        Intent launch = new Intent(Intent.ACTION_MAIN).addCategory(tv
                ? Intent.CATEGORY_LEANBACK_LAUNCHER : Intent.CATEGORY_LAUNCHER)
                .setPackage(activity.getPackageName());
        assertEquals(ViewerActivity.class.getName(), activity.getPackageManager()
                .queryIntentActivities(launch, 0).get(0).activityInfo.name);
        assertEquals(tv ? ActivityInfo.SCREEN_ORIENTATION_LANDSCAPE : ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED,
                activity.getRequestedOrientation());
        View root = activity.findViewById(android.R.id.content);
        assertTrue(hasButton(root, "Open viewer"));
        assertTrue(hasButton(root, "Parent mode"));
        assertTrue(hasButton(root, "Advanced server address"));
        assertEquals(tv, hasButton(root, "Pair using a phone"));
    }

    private boolean hasButton(View view, String label) {
        if (view instanceof Button) return label.contentEquals(((Button)view).getText());
        if (view instanceof ViewGroup) {
            ViewGroup group = (ViewGroup)view;
            for (int i = 0; i < group.getChildCount(); i++)
                if (hasButton(group.getChildAt(i), label)) return true;
        }
        return false;
    }

    @Test public void platformAndViewerAgreeOnTheLocalException() throws Exception {
        Resources resources = RuntimeEnvironment.getApplication().getResources();
        String local = resources.getString(R.string.local_test_origin);
        assertEquals(BuildConfig.DEBUG ? "http://192.168.86.163:8095" : "", local);
        int allowedHosts = 0;
        boolean baseDenied = false;
        try (XmlResourceParser policy = resources.getXml(R.xml.network_security_config)) {
            while (policy.next() != XmlPullParser.END_DOCUMENT) {
                if (policy.getEventType() != XmlPullParser.START_TAG) continue;
                switch (policy.getName()) {
                    case "base-config":
                        baseDenied = !policy.getAttributeBooleanValue(null, "cleartextTrafficPermitted", true);
                        break;
                    case "domain-config":
                        assertTrue(policy.getAttributeBooleanValue(null, "cleartextTrafficPermitted", false));
                        break;
                    case "domain":
                        assertFalse(policy.getAttributeBooleanValue(null, "includeSubdomains", true));
                        assertEquals(java.net.URI.create(local).getHost(), policy.nextText());
                        allowedHosts++;
                        break;
                    default: break;
                }
            }
        }
        assertTrue(baseDenied);
        assertEquals(BuildConfig.DEBUG ? 1 : 0, allowedHosts);
    }

    @Test public void localHttpRequiresExactConfiguredOrigin() {
        String local = "http://192.168.86.163:8095";
        assertEquals(local, ViewerActivity.origin(local + "/", local));
        assertNull(ViewerActivity.origin(local, ""));
        for (String value : new String[]{local + "/watch", local + "?q=x", local + "#secret",
                "http://192.168.86.163", "http://192.168.86.164:8095", "http://example.org",
                "http://192.168.86.163:8095@example.org", "https://user@example.org"}) {
            assertNull(value, ViewerActivity.origin(value, local));
        }
        assertEquals("https://tv.example", ViewerActivity.origin("https://tv.example/", ""));
    }
}
