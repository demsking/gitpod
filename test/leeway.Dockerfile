# Copyright (c) 2020 Gitpod GmbH. All rights reserved.
# Licensed under the GNU Affero General Public License (AGPL).
# See License-AGPL.txt in the project root for license information.

FROM alpine:latest
RUN apk add ca-certificates
# convenience scripting tools
RUN apk add bash moreutils
COPY test--app/bin /tests
ENV PATH=$PATH:/tests
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]
