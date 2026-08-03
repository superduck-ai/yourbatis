package com.superduck.yourbatis.idea.navigation;

import com.intellij.codeInsight.daemon.LineMarkerInfo;
import com.intellij.codeInsight.daemon.LineMarkerProviderDescriptor;
import com.intellij.codeInsight.navigation.NavigationGutterIconBuilder;
import com.intellij.icons.AllIcons;
import com.intellij.lang.ASTNode;
import com.intellij.openapi.editor.markup.GutterIconRenderer;
import com.intellij.psi.PsiElement;
import com.intellij.psi.xml.XmlTag;
import com.intellij.psi.xml.XmlToken;
import com.intellij.psi.xml.XmlTokenType;
import com.intellij.psi.util.PsiTreeUtil;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

import javax.swing.Icon;
import java.util.Collection;
import java.util.List;
import java.util.Set;

/** Adds independent gutter buttons for navigation from mapper XML to Go. */
public final class YourbatisGoLineMarkerProvider extends LineMarkerProviderDescriptor {
    static final String METHOD_TOOLTIP = "Navigate to Yourbatis Go mapper method";
    static final String MAPPER_TOOLTIP = "Navigate to Yourbatis Go mapper interface";

    private static final Set<String> STATEMENT_TAGS = Set.of(
            "select",
            "insert",
            "update",
            "delete"
    );

    @Override
    public @NotNull String getName() {
        return "Yourbatis Go navigation";
    }

    @Override
    public @NotNull Icon getIcon() {
        return AllIcons.Nodes.Interface;
    }

    @Override
    public @Nullable LineMarkerInfo<?> getLineMarkerInfo(@NotNull PsiElement element) {
        // Go stub-index access belongs in the slow line-marker pass.
        return null;
    }

    @Override
    public void collectSlowLineMarkers(
            @NotNull List<? extends PsiElement> elements,
            @NotNull Collection<? super LineMarkerInfo<?>> result
    ) {
        for (PsiElement element : elements) {
            XmlTag tag = navigationTagForStartName(element);
            if (tag == null || !isNavigationTag(tag)) {
                continue;
            }

            List<PsiElement> targets = YourbatisGoTargetResolver.findTargets(tag);
            if (targets.isEmpty()) {
                continue;
            }

            boolean mapper = "mapper".equals(tag.getLocalName());
            Icon icon = mapper ? AllIcons.Nodes.Interface : AllIcons.Nodes.Method;
            String tooltip = mapper ? MAPPER_TOOLTIP : METHOD_TOOLTIP;
            LineMarkerInfo<PsiElement> marker = NavigationGutterIconBuilder
                    .create(icon)
                    .setTargets(targets)
                    .setTooltipText(tooltip)
                    .setPopupTitle("Yourbatis Go Mapper")
                    .setAlignment(GutterIconRenderer.Alignment.LEFT)
                    .createLineMarkerInfo(element);
            result.add(marker);
        }
    }

    private static XmlTag navigationTagForStartName(PsiElement element) {
        if (!(element instanceof XmlToken token)
                || (token.getTokenType() != XmlTokenType.XML_TAG_NAME
                && token.getTokenType() != XmlTokenType.XML_NAME)) {
            return null;
        }
        XmlTag tag = PsiTreeUtil.getParentOfType(element, XmlTag.class, false);
        if (tag == null) {
            return null;
        }

        ASTNode nameNode = tag.getNode().findChildByType(XmlTokenType.XML_TAG_NAME);
        if (nameNode == null) {
            nameNode = tag.getNode().findChildByType(XmlTokenType.XML_NAME);
        }
        return nameNode != null && element.equals(nameNode.getPsi()) ? tag : null;
    }

    private static boolean isNavigationTag(XmlTag tag) {
        if ("mapper".equals(tag.getLocalName())) {
            return tag.getParentTag() == null;
        }
        return STATEMENT_TAGS.contains(tag.getLocalName())
                && tag.getParentTag() != null
                && "mapper".equals(tag.getParentTag().getLocalName());
    }
}
