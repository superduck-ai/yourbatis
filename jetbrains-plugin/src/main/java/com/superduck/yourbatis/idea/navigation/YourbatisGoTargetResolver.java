package com.superduck.yourbatis.idea.navigation;

import com.goide.psi.GoFile;
import com.goide.psi.GoInterfaceType;
import com.goide.psi.GoMethodSpec;
import com.goide.psi.GoSpecType;
import com.goide.psi.GoTypeSpec;
import com.goide.stubs.index.GoTypesIndex;
import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.psi.PsiElement;
import com.intellij.psi.PsiFile;
import com.intellij.psi.search.GlobalSearchScope;
import com.intellij.psi.xml.XmlTag;
import org.jetbrains.annotations.NotNull;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Comparator;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;

/** Resolves a Yourbatis mapper XML tag to its Go interface or method. */
final class YourbatisGoTargetResolver {
    private static final Set<String> STATEMENT_TAGS = Set.of(
            "select",
            "insert",
            "update",
            "delete"
    );

    private YourbatisGoTargetResolver() {
    }

    static @NotNull List<PsiElement> findTargets(@NotNull XmlTag tag) {
        XmlSource source = xmlSource(tag);
        if (source == null) {
            return List.of();
        }

        Collection<GoTypeSpec> indexedTypes = GoTypesIndex.find(
                source.mapperName(),
                tag.getProject(),
                GlobalSearchScope.projectScope(tag.getProject()),
                null
        );
        List<GoTypeSpec> mapperTypes = indexedTypes
                .stream()
                .filter(YourbatisGoTargetResolver::isInterface)
                .sorted(mapperOrder(source.namespace(), source.originFile()))
                .toList();

        if (source.statementId() == null) {
            return new ArrayList<>(mapperTypes);
        }

        LinkedHashSet<PsiElement> methods = new LinkedHashSet<>();
        for (GoTypeSpec mapperType : mapperTypes) {
            GoInterfaceType interfaceType = interfaceType(mapperType);
            if (interfaceType == null) {
                continue;
            }
            for (GoMethodSpec method : interfaceType.getAllMethods(tag)) {
                if (source.statementId().equals(method.getName())) {
                    methods.add(method);
                }
            }
        }
        return new ArrayList<>(methods);
    }

    private static XmlSource xmlSource(XmlTag tag) {
        XmlTag mapperTag;
        String statementId = null;
        if ("mapper".equals(tag.getLocalName()) && tag.getParentTag() == null) {
            mapperTag = tag;
        } else if (STATEMENT_TAGS.contains(tag.getLocalName())
                && tag.getParentTag() != null
                && "mapper".equals(tag.getParentTag().getLocalName())) {
            mapperTag = tag.getParentTag();
            statementId = trimmedOrNull(tag.getAttributeValue("id"));
            if (statementId == null) {
                return null;
            }
        } else {
            return null;
        }

        String namespace = trimmedOrNull(mapperTag.getAttributeValue("namespace"));
        if (namespace == null) {
            return null;
        }
        String mapperName = simpleMapperName(namespace);
        if (mapperName == null) {
            return null;
        }
        PsiFile file = tag.getContainingFile();
        VirtualFile originFile = file == null ? null : file.getVirtualFile();
        return new XmlSource(namespace, mapperName, statementId, originFile);
    }

    private static GoInterfaceType interfaceType(GoTypeSpec typeSpec) {
        GoSpecType specType = typeSpec.getSpecType();
        return specType != null && specType.getType() instanceof GoInterfaceType interfaceType
                ? interfaceType
                : null;
    }

    private static boolean isInterface(GoTypeSpec typeSpec) {
        return interfaceType(typeSpec) != null;
    }

    private static Comparator<GoTypeSpec> mapperOrder(String namespace, VirtualFile originFile) {
        String qualifier = namespaceQualifier(namespace);
        VirtualFile originDirectory = originFile == null ? null : originFile.getParent();
        return Comparator
                .comparingInt((GoTypeSpec candidate) -> namespaceRank(candidate, qualifier))
                .thenComparingInt(candidate -> sameDirectoryRank(candidate, originDirectory))
                .thenComparing(YourbatisGoTargetResolver::containingPath);
    }

    private static int namespaceRank(GoTypeSpec candidate, String qualifier) {
        if (qualifier == null) {
            return 0;
        }
        PsiFile file = candidate.getContainingFile();
        if (file instanceof GoFile goFile && qualifier.equals(goFile.getImportPath(false))) {
            return 0;
        }
        return 1;
    }

    private static int sameDirectoryRank(GoTypeSpec candidate, VirtualFile originDirectory) {
        PsiFile file = candidate.getContainingFile();
        VirtualFile virtualFile = file == null ? null : file.getVirtualFile();
        return originDirectory != null
                && virtualFile != null
                && originDirectory.equals(virtualFile.getParent()) ? 0 : 1;
    }

    private static String containingPath(GoTypeSpec candidate) {
        PsiFile file = candidate.getContainingFile();
        VirtualFile virtualFile = file == null ? null : file.getVirtualFile();
        return virtualFile == null ? "" : virtualFile.getPath();
    }

    private static String namespaceQualifier(String namespace) {
        int lastDot = namespace.lastIndexOf('.');
        return lastDot < 0 ? null : trimmedOrNull(namespace.substring(0, lastDot));
    }

    private static String simpleMapperName(String namespace) {
        int lastDot = namespace.lastIndexOf('.');
        return lastDot < 0 ? namespace : trimmedOrNull(namespace.substring(lastDot + 1));
    }

    private static String trimmedOrNull(String value) {
        if (value == null) {
            return null;
        }
        String trimmed = value.trim();
        return trimmed.isEmpty() ? null : trimmed;
    }

    private record XmlSource(
            String namespace,
            String mapperName,
            String statementId,
            VirtualFile originFile
    ) {
    }
}
