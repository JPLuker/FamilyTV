package com.jpluker.familytv;

import android.app.Activity;
import android.content.pm.ActivityInfo;
import android.graphics.Insets;
import android.os.Build;
import android.view.View;
import android.view.WindowInsets;

final class ViewerWindow {
    private ViewerWindow() {}

    static void configure(Activity activity) {
        if (activity.getResources().getBoolean(R.bool.television)) {
            activity.setRequestedOrientation(ActivityInfo.SCREEN_ORIENTATION_LANDSCAPE);
        }
        if (Build.VERSION.SDK_INT >= 35) {
            View content = activity.findViewById(android.R.id.content);
            content.setOnApplyWindowInsetsListener((view, insets) -> {
                Insets edges = insets.getInsets(WindowInsets.Type.systemBars()
                        | WindowInsets.Type.displayCutout() | WindowInsets.Type.ime());
                view.setPadding(edges.left, edges.top, edges.right, edges.bottom);
                return WindowInsets.CONSUMED;
            });
        }
    }
}
