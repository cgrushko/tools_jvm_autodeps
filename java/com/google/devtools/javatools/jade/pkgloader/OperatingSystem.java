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

import java.util.Locale;
import java.util.logging.Logger;

enum OperatingSystem {
  LINUX,
  MACOS,
  OTHER;

  private static final Logger logger = Logger.getLogger("OperatingSystem");

  static OperatingSystem detect() {
    final String name;
    try {
      name = System.getProperty("os.name").toLowerCase(Locale.UK).trim();
    } catch (Exception e) {
      logger.warning("Can't detect current OS, assuming 'other'.");
      return OTHER;
    }
    if (name.startsWith("mac") || name.contains("bsd") || name.startsWith("darwin")) {
      return MACOS;
    }

    if (name.startsWith("linux")) {
      return LINUX;
    }

    return OTHER;
  }
}
