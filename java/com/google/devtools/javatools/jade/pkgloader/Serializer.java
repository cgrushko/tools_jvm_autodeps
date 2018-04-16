// Copyright 2018 The Jadep Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.google.devtools.javatools.jade.pkgloader;

import static com.google.devtools.build.lib.packages.BuildType.LABEL;
import static com.google.devtools.build.lib.packages.BuildType.LABEL_LIST;
import static com.google.devtools.build.lib.packages.BuildType.NODEP_LABEL;
import static com.google.devtools.build.lib.packages.BuildType.NODEP_LABEL_LIST;
import static com.google.devtools.build.lib.packages.BuildType.OUTPUT;
import static com.google.devtools.build.lib.packages.BuildType.OUTPUT_LIST;
import static com.google.devtools.build.lib.packages.BuildType.TRISTATE;
import static com.google.devtools.build.lib.syntax.Type.BOOLEAN;
import static com.google.devtools.build.lib.syntax.Type.INTEGER;
import static com.google.devtools.build.lib.syntax.Type.STRING;
import static com.google.devtools.build.lib.syntax.Type.STRING_LIST;

import com.google.devtools.build.lib.cmdline.Label;
import com.google.devtools.build.lib.cmdline.PackageIdentifier;
import com.google.devtools.build.lib.packages.Attribute.ComputedDefault;
import com.google.devtools.build.lib.packages.BuildType.Selector;
import com.google.devtools.build.lib.packages.BuildType.SelectorList;
import com.google.devtools.build.lib.packages.InputFile;
import com.google.devtools.build.lib.packages.OutputFile;
import com.google.devtools.build.lib.packages.Package;
import com.google.devtools.build.lib.packages.PackageGroup;
import com.google.devtools.build.lib.packages.RawAttributeMapper;
import com.google.devtools.build.lib.packages.Rule;
import com.google.devtools.build.lib.packages.TriState;
import com.google.devtools.build.lib.syntax.Type;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.messages.Messages;
import com.google.protos.java.com.google.devtools.javatools.jade.pkgloader.messages.Messages.Attribute;
import java.util.Collection;
import java.util.List;
import java.util.Set;
import java.util.logging.Level;
import java.util.logging.Logger;

/** Serializes a Bazel Package object into our proto. */
public class Serializer {
  private static final Logger logger = Logger.getLogger(Serializer.class.getName());

  private static boolean shouldSerializeImplicitAttribute(String name) {
    switch (name) {
      case "testonly":
      case "visibility":
      case "deprecation":
        return true;
      default:
        return false;
    }
  }

  static Messages.Pkg serialize(Package pkg, Set<String> ruleKindsToSerialize) {
    Messages.Pkg.Builder result = Messages.Pkg.newBuilder();

    result.setPath(pkg.getPackageDirectory().getPathString());

    for (Label label : pkg.getDefaultVisibility().getDeclaredLabels()) {
      result.addDefaultVisibility(label.toString());
    }

    pkg.getTargets()
        .forEach(
            (name, target) -> {
              if (target instanceof InputFile) {
                result.putFiles(name, "");
              } else if (target instanceof OutputFile) {
                result.putFiles(name, ((OutputFile) target).getGeneratingRule().getName());
              } else if (target instanceof Rule) {
                if (!ruleKindsToSerialize.isEmpty()
                    && !ruleKindsToSerialize.contains(((Rule) target).getRuleClass())) {
                  return;
                }
                try {
                  result.putRules(name, serializeRule((Rule) target));
                } catch (Exception e) {
                  logger.log(
                      Level.WARNING,
                      String.format("Failed to serialize rule %s, skipping.", target.getLabel()),
                      e);
                }
              } else if (target instanceof PackageGroup) {
                result.putPackageGroups(name, serializePackage((PackageGroup) target));
              }
            });
    return result.build();
  }

  static Messages.Rule serializeRule(Rule rule) {
    Messages.Rule.Builder result = Messages.Rule.newBuilder();

    result.setKind(stripSuffix(rule.getTargetKind(), " rule"));
    Package currentPkg = rule.getPackage();
    result.setPosition(
        String.format(
            "%s/BUILD:%d:%d",
            currentPkg.getName(),
            rule.getLocation().getStartLine(),
            rule.getLocation().getStartLineAndColumn().getColumn()));

    RawAttributeMapper rawAttributeMapper = RawAttributeMapper.of(rule);
    for (com.google.devtools.build.lib.packages.Attribute attr : rule.getAttributes()) {
      serializeAttribute(rule, result, rawAttributeMapper, attr);
    }

    return result.build();
  }

  private static void serializeAttribute(
      Rule rule,
      Messages.Rule.Builder result,
      RawAttributeMapper rawAttributeMapper,
      com.google.devtools.build.lib.packages.Attribute attr) {
    Object rawAttributeValue = rawAttributeMapper.getRawAttributeValue(rule, attr);
    String name = attr.getName();
    Type<?> type = attr.getType();

    if (!rule.isAttributeValueExplicitlySpecified(attr)
        && !shouldSerializeImplicitAttribute(name)) {
      return;
    }

    Object value = rawAttributeValue;
    if (rawAttributeValue instanceof ComputedDefault) {
      value = rawAttributeMapper.get(name, type);
    }
    if (value == null) {
      return;
    }

    if (type == LABEL_LIST || type == NODEP_LABEL_LIST || type == OUTPUT_LIST) {
      Attribute.Builder a = Attribute.newBuilder();
      Messages.Strings.Builder strs = a.getListOfStringsBuilder();
      PackageIdentifier pkgId = rule.getPackage().getPackageIdentifier();
      if (value instanceof SelectorList<?>) {
        // Since type is a ListType<Label>, value must be rawValue + select(...) + select(...) + ...
        // where each element is a list of labels.
        @SuppressWarnings("unchecked")
        SelectorList<List<Label>> selList = (SelectorList<List<Label>>) value;

        for (Selector<? extends Collection<Label>> selector :      selList.getSelectors()) {
          for (Collection<Label> vs : selector.getEntries().values()) {
            for (Label lbl : vs) {
              strs.addStr(serializeLabel(pkgId, name, lbl));
            }
          }
        }
      } else {
        for (Label entry : LABEL_LIST.cast(value)) {
          strs.addStr(serializeLabel(pkgId, name, entry));
        }
      }
      result.putAttributes(name, a.build());
      return;
    }

    if (value instanceof SelectorList<?>) {
      List<? extends Selector<?>> selectors = ((SelectorList<?>) value).getSelectors();
      if (selectors.size() == 1 && selectors.get(0).hasDefault()) {
        value = selectors.get(0).getDefault();
      } else {
        result.putAttributes(attr.getName(), Attribute.newBuilder().setUnknown(true).build());
        return;
      }
    }

    if (type == INTEGER) {
      result.putAttributes(name, Attribute.newBuilder().setI((Integer) value).build());
    } else if (type == STRING) {
      result.putAttributes(name, Attribute.newBuilder().setS((String) value).build());
    } else if (type == LABEL || type == NODEP_LABEL || type == OUTPUT) {
      result.putAttributes(
          name,
          Attribute.newBuilder()
              .setS(serializeLabel(rule.getPackage().getPackageIdentifier(), name, (Label) value))
              .build());
    } else if (type == STRING_LIST) {
      Attribute.Builder a = Attribute.newBuilder();
      Messages.Strings.Builder strs = a.getListOfStringsBuilder();
      for (Object entry : (Collection<?>) value) {
        strs.addStr(entry.toString());
      }
      result.putAttributes(name, a.build());
    } else if (type == BOOLEAN) {
      result.putAttributes(name, Attribute.newBuilder().setB((Boolean) value).build());
    } else if (type == TRISTATE) {
      result.putAttributes(name, Attribute.newBuilder().setI(((TriState) value).toInt()).build());
    } else {
      result.putAttributes(attr.getName(), Attribute.newBuilder().setUnknown(true).build());
    }
  }

  private static String serializeLabel(
      PackageIdentifier containingPackage, String attrName, Label label) {
    // TODO: remove this once Jadep handles relative visibility labels.
    if (attrName.equals("visibility")) {
      return label.getCanonicalForm();
    }

    if (containingPackage.equals(label.getPackageIdentifier())) {
      return label.getName();
    }
    return label.getCanonicalForm();
  }

  static Messages.PackageGroup serializePackage(PackageGroup group) {
    Messages.PackageGroup.Builder result = Messages.PackageGroup.newBuilder();
    group.getPackageSpecifications().containedPackages().forEachOrdered(result::addPackageSpecs);
    for (Label include : group.getIncludes()) {
      result.addIncludes(include.toString());
    }
    return result.build();
  }

  private static String stripSuffix(String s, String suffix) {
    if (s.endsWith(suffix)) {
      return s.substring(0, s.length() - suffix.length());
    }
    return s;
  }
}
