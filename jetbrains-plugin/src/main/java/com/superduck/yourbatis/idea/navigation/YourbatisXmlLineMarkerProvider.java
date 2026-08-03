package com.superduck.yourbatis.idea.navigation;

import com.goide.psi.GoMethodSpec;
import com.goide.psi.GoSpecType;
import com.goide.psi.GoTypeSpec;
import com.intellij.codeInsight.daemon.LineMarkerInfo;
import com.intellij.codeInsight.daemon.LineMarkerProviderDescriptor;
import com.intellij.codeInsight.navigation.NavigationGutterIconBuilder;
import com.intellij.icons.AllIcons;
import com.intellij.openapi.editor.markup.GutterIconRenderer;
import com.intellij.psi.PsiElement;
import com.intellij.psi.xml.XmlTag;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

import javax.swing.Icon;
import java.util.Collection;
import java.util.List;

/** Adds an independent gutter button for navigation to Yourbatis mapper XML. */
public final class YourbatisXmlLineMarkerProvider extends LineMarkerProviderDescriptor {
    static final String METHOD_TOOLTIP = "Navigate to Yourbatis XML statement";
    static final String MAPPER_TOOLTIP = "Navigate to Yourbatis mapper XML";

    private static final Icon ICON = AllIcons.FileTypes.Xml;

    @Override
    public @NotNull String getName() {
        return "Yourbatis XML navigation";
    }

    @Override
    public @NotNull Icon getIcon() {
        return ICON;
    }

    @Override
    public @Nullable LineMarkerInfo<?> getLineMarkerInfo(@NotNull PsiElement element) {
        // Index access belongs in the slow line-marker pass.
        return null;
    }

    @Override
    public void collectSlowLineMarkers(
            @NotNull List<? extends PsiElement> elements,
            @NotNull Collection<? super LineMarkerInfo<?>> result
    ) {
        for (PsiElement element : elements) {
            PsiElement source = navigationSourceForIdentifier(element);
            if (source == null) {
                continue;
            }

            List<XmlTag> targets = YourbatisXmlTargetResolver.findTargets(source);
            if (targets.isEmpty()) {
                continue;
            }

            String tooltip = source instanceof GoMethodSpec ? METHOD_TOOLTIP : MAPPER_TOOLTIP;
            LineMarkerInfo<PsiElement> marker = NavigationGutterIconBuilder
                    .create(ICON)
                    .setTargets(targets)
                    .setTooltipText(tooltip)
                    .setPopupTitle("Yourbatis Mapper XML")
                    // Go's implementation marker is right-aligned. Keeping this
                    // one on the left gives both actions their own gutter button.
                    .setAlignment(GutterIconRenderer.Alignment.LEFT)
                    .createLineMarkerInfo(element);
            result.add(marker);
        }
    }

    private static PsiElement navigationSourceForIdentifier(PsiElement element) {
        PsiElement parent = element.getParent();
        if (parent instanceof GoMethodSpec method && element.equals(method.getIdentifier())) {
            return method;
        }
        if (parent instanceof GoSpecType specType
                && element.equals(specType.getIdentifier())
                && specType.getParent() instanceof GoTypeSpec typeSpec) {
            return typeSpec;
        }
        return null;
    }
}
