package com.superduck.yourbatis.idea.navigation;

import com.goide.GoFileType;
import com.intellij.codeInsight.daemon.GutterMark;
import com.intellij.openapi.editor.markup.GutterIconRenderer;
import com.intellij.psi.PsiElement;
import com.intellij.psi.search.searches.DefinitionsScopedSearch;
import com.intellij.psi.xml.XmlTag;
import com.intellij.testFramework.fixtures.BasePlatformTestCase;

import java.util.Collection;
import java.util.List;

public class YourbatisXmlLineMarkerProviderTest extends BasePlatformTestCase {
    private static final String MAPPER_XML = """
            <mapper namespace="github.com/acme/user.UserMapper">
                <select id="GetByID">SELECT id FROM users WHERE id = #{id}</select>
                <insert id="Create">INSERT INTO users DEFAULT VALUES</insert>
            </mapper>
            """;

    public void testMethodHasIndependentXmlGutterMarker() {
        myFixture.addFileToProject("user_mapper.xml", MAPPER_XML);
        myFixture.configureByText(GoFileType.INSTANCE, """
                package user

                type UserMapper interface {
                    Get<caret>ByID(id int64) (int64, error)
                }
                """);

        GutterIconRenderer renderer = findMarkerAtCaret(
                YourbatisXmlLineMarkerProvider.METHOD_TOOLTIP
        );
        List<XmlTag> targets = targetsAtCaret();
        XmlTag statement = findXmlTag(targets, "select");
        assertNotNull(targets.toString(), statement);
        assertEquals("GetByID", statement.getAttributeValue("id"));
        assertEquals(GutterIconRenderer.Alignment.LEFT, renderer.getAlignment());
        assertTrue(renderer.isNavigateAction());
    }

    public void testInterfaceHasMapperXmlGutterMarker() {
        myFixture.addFileToProject("user_mapper.xml", MAPPER_XML);
        myFixture.configureByText(GoFileType.INSTANCE, """
                package user

                type User<caret>Mapper interface {
                    GetByID(id int64) (int64, error)
                }
                """);

        GutterIconRenderer renderer = findMarkerAtCaret(
                YourbatisXmlLineMarkerProvider.MAPPER_TOOLTIP
        );
        List<XmlTag> targets = targetsAtCaret();
        XmlTag mapper = findXmlTag(targets, "mapper");
        assertNotNull(targets.toString(), mapper);
        assertEquals("github.com/acme/user.UserMapper", mapper.getAttributeValue("namespace"));
    }

    public void testUnrelatedInterfaceHasNoXmlGutterMarker() {
        myFixture.addFileToProject("user_mapper.xml", MAPPER_XML);
        myFixture.configureByText(GoFileType.INSTANCE, """
                package user

                type OtherMapper interface {
                    Get<caret>ByID(id int64) (int64, error)
                }
                """);

        myFixture.doHighlighting();
        assertNull(findMarker(YourbatisXmlLineMarkerProvider.METHOD_TOOLTIP));
    }

    public void testPluginDoesNotOverrideGoToImplementation() {
        boolean hasYourbatisDefinitionsSearcher = DefinitionsScopedSearch.EP_NAME
                .getExtensionList()
                .stream()
                .anyMatch(searcher -> searcher.getClass().getName().startsWith(
                        "com.superduck.yourbatis.idea.navigation"
                ));

        assertFalse(DefinitionsScopedSearch.EP_NAME.getExtensionList().toString(), hasYourbatisDefinitionsSearcher);
    }

    private GutterIconRenderer findMarkerAtCaret(String tooltip) {
        myFixture.doHighlighting();
        GutterMark marker = findMarker(tooltip);
        assertNotNull(myFixture.findGuttersAtCaret().toString(), marker);
        assertInstanceOf(marker, GutterIconRenderer.class);
        return (GutterIconRenderer) marker;
    }

    private List<XmlTag> targetsAtCaret() {
        PsiElement identifier = myFixture.getFile().findElementAt(myFixture.getCaretOffset());
        assertNotNull(identifier);
        return YourbatisXmlTargetResolver.findTargets(identifier);
    }

    private GutterMark findMarker(String tooltip) {
        return myFixture.findGuttersAtCaret()
                .stream()
                .filter(marker -> tooltip.equals(marker.getTooltipText()))
                .findFirst()
                .orElse(null);
    }

    private static XmlTag findXmlTag(Collection<? extends PsiElement> elements, String localName) {
        return elements.stream()
                .filter(XmlTag.class::isInstance)
                .map(XmlTag.class::cast)
                .filter(tag -> localName.equals(tag.getLocalName()))
                .findFirst()
                .orElse(null);
    }
}
