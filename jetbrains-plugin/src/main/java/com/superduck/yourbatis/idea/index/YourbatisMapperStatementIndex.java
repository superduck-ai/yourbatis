package com.superduck.yourbatis.idea.index;

import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.util.indexing.DataIndexer;
import com.intellij.util.indexing.FileBasedIndex;
import com.intellij.util.indexing.FileContent;
import com.intellij.util.indexing.ID;
import com.intellij.util.indexing.ScalarIndexExtension;
import com.intellij.util.io.EnumeratorStringDescriptor;
import com.intellij.util.io.KeyDescriptor;
import org.jetbrains.annotations.NotNull;

import javax.xml.stream.XMLInputFactory;
import javax.xml.stream.XMLStreamConstants;
import javax.xml.stream.XMLStreamException;
import javax.xml.stream.XMLStreamReader;
import java.io.StringReader;
import java.util.Collections;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

/**
 * Maps a simple mapper namespace and statement ID to the XML files containing it.
 * PSI is deliberately not used while indexing.
 */
public final class YourbatisMapperStatementIndex extends ScalarIndexExtension<String> {
    public static final ID<String, Void> NAME =
            ID.create("com.superduck.yourbatis.mapperStatement");

    public static final String MAPPER_ROOT = "<mapper>";

    private static final char KEY_SEPARATOR = '\u001f';
    private static final int VERSION = 1;
    private static final Set<String> STATEMENT_TAGS = Set.of(
            "select",
            "insert",
            "update",
            "delete"
    );

    private static final ThreadLocal<XMLInputFactory> XML_INPUT_FACTORY =
            ThreadLocal.withInitial(YourbatisMapperStatementIndex::createXmlInputFactory);

    @Override
    public @NotNull ID<String, Void> getName() {
        return NAME;
    }

    @Override
    public @NotNull DataIndexer<String, Void, FileContent> getIndexer() {
        return inputData -> extractKeys(inputData.getContentAsText());
    }

    @Override
    public @NotNull KeyDescriptor<String> getKeyDescriptor() {
        return EnumeratorStringDescriptor.INSTANCE;
    }

    @Override
    public @NotNull FileBasedIndex.InputFilter getInputFilter() {
        return YourbatisMapperStatementIndex::isXmlFile;
    }

    @Override
    public boolean dependsOnFileContent() {
        return true;
    }

    @Override
    public int getVersion() {
        return VERSION;
    }

    @Override
    public boolean traceKeyHashToVirtualFileMapping() {
        return true;
    }

    public static @NotNull String key(@NotNull String mapperName, @NotNull String statementId) {
        return mapperName + KEY_SEPARATOR + statementId;
    }

    static @NotNull Map<String, Void> extractKeys(@NotNull CharSequence content) {
        Map<String, Void> result = new HashMap<>();
        XMLStreamReader reader = null;
        try {
            reader = XML_INPUT_FACTORY.get().createXMLStreamReader(new StringReader(content.toString()));
            int depth = 0;
            String mapperName = null;

            while (reader.hasNext()) {
                int event = reader.next();
                if (event == XMLStreamConstants.START_ELEMENT) {
                    depth++;
                    String localName = reader.getLocalName();
                    if (depth == 1 && "mapper".equals(localName)) {
                        mapperName = simpleMapperName(reader.getAttributeValue(null, "namespace"));
                        if (mapperName != null) {
                            result.put(key(mapperName, MAPPER_ROOT), null);
                        }
                    } else if (depth == 2 && mapperName != null && STATEMENT_TAGS.contains(localName)) {
                        String statementId = trimmedOrNull(reader.getAttributeValue(null, "id"));
                        if (statementId != null) {
                            result.put(key(mapperName, statementId), null);
                        }
                    }
                } else if (event == XMLStreamConstants.END_ELEMENT) {
                    depth--;
                }
            }
            return result;
        } catch (XMLStreamException ignored) {
            return Collections.emptyMap();
        } finally {
            if (reader != null) {
                try {
                    reader.close();
                } catch (XMLStreamException ignored) {
                    // Nothing to release beyond parser-local state.
                }
            }
        }
    }

    private static boolean isXmlFile(@NotNull VirtualFile file) {
        return "xml".equalsIgnoreCase(file.getExtension());
    }

    private static XMLInputFactory createXmlInputFactory() {
        XMLInputFactory factory = XMLInputFactory.newFactory();
        setPropertyIfSupported(factory, XMLInputFactory.SUPPORT_DTD, false);
        setPropertyIfSupported(factory, "javax.xml.stream.isSupportingExternalEntities", false);
        setPropertyIfSupported(factory, XMLInputFactory.IS_REPLACING_ENTITY_REFERENCES, false);
        return factory;
    }

    private static void setPropertyIfSupported(XMLInputFactory factory, String property, boolean value) {
        try {
            factory.setProperty(property, value);
        } catch (IllegalArgumentException ignored) {
            // The JDK parser supports these flags; tolerate alternate platform parsers.
        }
    }

    private static String simpleMapperName(String namespace) {
        String normalized = trimmedOrNull(namespace);
        if (normalized == null) {
            return null;
        }
        int lastDot = normalized.lastIndexOf('.');
        return lastDot >= 0 ? trimmedOrNull(normalized.substring(lastDot + 1)) : normalized;
    }

    private static String trimmedOrNull(String value) {
        if (value == null) {
            return null;
        }
        String trimmed = value.trim();
        return trimmed.isEmpty() ? null : trimmed;
    }
}
