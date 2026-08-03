package com.superduck.yourbatis.idea.navigation;

import com.goide.psi.GoInterfaceType;
import com.goide.psi.GoMethodSpec;
import com.goide.psi.GoSpecType;
import com.goide.psi.GoTypeSpec;
import com.intellij.openapi.progress.ProgressManager;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.vfs.VirtualFile;
import com.intellij.psi.PsiElement;
import com.intellij.psi.PsiFile;
import com.intellij.psi.PsiManager;
import com.intellij.psi.search.GlobalSearchScope;
import com.intellij.psi.util.PsiTreeUtil;
import com.intellij.psi.xml.XmlFile;
import com.intellij.psi.xml.XmlTag;
import com.intellij.util.indexing.FileBasedIndex;
import org.jetbrains.annotations.NotNull;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Comparator;
import java.util.List;
import java.util.Set;

import static com.superduck.yourbatis.idea.index.YourbatisMapperStatementIndex.MAPPER_ROOT;
import static com.superduck.yourbatis.idea.index.YourbatisMapperStatementIndex.NAME;
import static com.superduck.yourbatis.idea.index.YourbatisMapperStatementIndex.key;

/** Resolves a Yourbatis Go mapper interface or method to matching mapper XML tags. */
final class YourbatisXmlTargetResolver {
    private static final Set<String> STATEMENT_TAGS = Set.of(
            "select",
            "insert",
            "update",
            "delete"
    );

    private YourbatisXmlTargetResolver() {
    }

    static @NotNull List<XmlTag> findTargets(@NotNull PsiElement element) {
        NavigationSource source = navigationSource(element);
        if (source == null) {
            return List.of();
        }

        Project project = element.getProject();
        GlobalSearchScope scope = GlobalSearchScope.projectScope(project);
        Collection<VirtualFile> indexedFiles = FileBasedIndex.getInstance().getContainingFiles(
                NAME,
                key(source.mapperName(), source.statementId() == null ? MAPPER_ROOT : source.statementId()),
                scope
        );

        List<VirtualFile> candidateFiles = new ArrayList<>(indexedFiles);
        candidateFiles.sort(candidateOrder(source.originFile()));

        PsiManager psiManager = PsiManager.getInstance(project);
        List<XmlTag> targets = new ArrayList<>(candidateFiles.size());
        for (VirtualFile candidateFile : candidateFiles) {
            ProgressManager.checkCanceled();
            PsiFile psiFile = psiManager.findFile(candidateFile);
            if (!(psiFile instanceof XmlFile xmlFile)) {
                continue;
            }

            XmlTag target = findTarget(xmlFile, source.mapperName(), source.statementId());
            if (target != null) {
                targets.add(target);
            }
        }
        return targets;
    }

    private static NavigationSource navigationSource(PsiElement element) {
        GoMethodSpec method = PsiTreeUtil.getParentOfType(element, GoMethodSpec.class, false);
        if (method != null && method.getIdentifier() != null) {
            GoTypeSpec mapperType = PsiTreeUtil.getParentOfType(method, GoTypeSpec.class);
            if (isInterface(mapperType)) {
                String mapperName = mapperType.getName();
                String methodName = method.getName();
                if (mapperName != null && methodName != null) {
                    return new NavigationSource(mapperName, methodName, originFile(mapperType));
                }
            }
            return null;
        }

        GoTypeSpec mapperType = PsiTreeUtil.getParentOfType(element, GoTypeSpec.class, false);
        if (!isInterface(mapperType)) {
            return null;
        }
        String mapperName = mapperType.getName();
        return mapperName == null ? null : new NavigationSource(mapperName, null, originFile(mapperType));
    }

    private static boolean isInterface(GoTypeSpec typeSpec) {
        if (typeSpec == null) {
            return false;
        }
        GoSpecType specType = typeSpec.getSpecType();
        return specType != null && specType.getType() instanceof GoInterfaceType;
    }

    private static VirtualFile originFile(GoTypeSpec typeSpec) {
        PsiFile file = typeSpec.getContainingFile();
        return file == null ? null : file.getVirtualFile();
    }

    private static Comparator<VirtualFile> candidateOrder(VirtualFile originFile) {
        VirtualFile originDirectory = originFile == null ? null : originFile.getParent();
        return Comparator
                .comparingInt((VirtualFile candidate) ->
                        originDirectory != null && originDirectory.equals(candidate.getParent()) ? 0 : 1)
                .thenComparing(VirtualFile::getPath);
    }

    private static XmlTag findTarget(XmlFile xmlFile, String mapperName, String statementId) {
        XmlTag root = xmlFile.getRootTag();
        if (root == null || !"mapper".equals(root.getLocalName())) {
            return null;
        }
        if (!matchesMapper(root.getAttributeValue("namespace"), mapperName)) {
            return null;
        }
        if (statementId == null) {
            return root;
        }

        for (XmlTag child : root.getSubTags()) {
            if (STATEMENT_TAGS.contains(child.getLocalName())
                    && statementId.equals(child.getAttributeValue("id"))) {
                return child;
            }
        }
        return null;
    }

    private static boolean matchesMapper(String namespace, String mapperName) {
        if (namespace == null) {
            return false;
        }
        String normalized = namespace.trim();
        return normalized.equals(mapperName) || normalized.endsWith("." + mapperName);
    }

    private record NavigationSource(String mapperName, String statementId, VirtualFile originFile) {
    }
}
