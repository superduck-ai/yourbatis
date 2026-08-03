package com.superduck.yourbatis.idea.navigation;

import com.goide.psi.GoMethodSpec;
import com.goide.psi.GoTypeSpec;
import com.intellij.codeInsight.daemon.GutterMark;
import com.intellij.ide.highlighter.XmlFileType;
import com.intellij.openapi.editor.markup.GutterIconRenderer;
import com.intellij.psi.PsiElement;
import com.intellij.psi.util.PsiTreeUtil;
import com.intellij.psi.xml.XmlTag;
import com.intellij.testFramework.fixtures.BasePlatformTestCase;

import java.util.List;

public class YourbatisGoLineMarkerProviderTest extends BasePlatformTestCase {
    private static final String GO_MAPPER = """
            package user

            type UserMapper interface {
                GetByID(id int64) (int64, error)
                Create() (int64, error)
            }
            """;

    public void testMapperTagNavigatesToGoInterface() {
        myFixture.addFileToProject("user_mapper.go", GO_MAPPER);
        myFixture.configureByText(XmlFileType.INSTANCE, """
                <map<caret>per namespace="github.com/acme/user.UserMapper">
                    <select id="GetByID">SELECT 1</select>
                </mapper>
                """);

        GutterIconRenderer renderer = findMarkerAtCaret(YourbatisGoLineMarkerProvider.MAPPER_TOOLTIP);
        List<PsiElement> targets = targetsAtCaret();
        GoTypeSpec mapper = targets.stream()
                .filter(GoTypeSpec.class::isInstance)
                .map(GoTypeSpec.class::cast)
                .findFirst()
                .orElse(null);
        assertNotNull(targets.toString(), mapper);
        assertEquals("UserMapper", mapper.getName());
        assertEquals(GutterIconRenderer.Alignment.LEFT, renderer.getAlignment());
        assertTrue(renderer.isNavigateAction());
    }

    public void testStatementTagNavigatesToGoInterfaceMethod() {
        myFixture.addFileToProject("user_mapper.go", GO_MAPPER);
        myFixture.configureByText(XmlFileType.INSTANCE, """
                <mapper namespace="UserMapper">
                    <sel<caret>ect id="GetByID">SELECT 1</select>
                </mapper>
                """);

        GutterIconRenderer renderer = findMarkerAtCaret(YourbatisGoLineMarkerProvider.METHOD_TOOLTIP);
        List<PsiElement> targets = targetsAtCaret();
        GoMethodSpec method = targets.stream()
                .filter(GoMethodSpec.class::isInstance)
                .map(GoMethodSpec.class::cast)
                .findFirst()
                .orElse(null);
        assertNotNull(targets.toString(), method);
        assertEquals("GetByID", method.getName());
        assertTrue(renderer.isNavigateAction());
    }

    public void testUnknownStatementHasNoGoGutterMarker() {
        myFixture.addFileToProject("user_mapper.go", GO_MAPPER);
        myFixture.configureByText(XmlFileType.INSTANCE, """
                <mapper namespace="UserMapper">
                    <sel<caret>ect id="DoesNotExist">SELECT 1</select>
                </mapper>
                """);

        myFixture.doHighlighting();
        assertNull(findMarker(YourbatisGoLineMarkerProvider.METHOD_TOOLTIP));
    }

    private GutterIconRenderer findMarkerAtCaret(String tooltip) {
        myFixture.doHighlighting();
        GutterMark marker = findMarker(tooltip);
        assertNotNull(myFixture.findGuttersAtCaret().toString(), marker);
        assertInstanceOf(marker, GutterIconRenderer.class);
        return (GutterIconRenderer) marker;
    }

    private GutterMark findMarker(String tooltip) {
        return myFixture.findGuttersAtCaret()
                .stream()
                .filter(marker -> tooltip.equals(marker.getTooltipText()))
                .findFirst()
                .orElse(null);
    }

    private List<PsiElement> targetsAtCaret() {
        PsiElement token = myFixture.getFile().findElementAt(myFixture.getCaretOffset());
        assertNotNull(token);
        XmlTag tag = PsiTreeUtil.getParentOfType(token, XmlTag.class, false);
        assertNotNull(tag);
        return YourbatisGoTargetResolver.findTargets(tag);
    }
}
