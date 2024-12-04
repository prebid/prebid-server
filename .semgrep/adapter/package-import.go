import (
	// ok: package-import-check
	"fmt"
	// ok: package-import-check
	"os"
	// ruleid: package-import-check
	"github.com/mitchellh/copystructure"
	// ruleid: package-import-check
	"github.com/prebid/prebid-server/v3/di"
)

import (
	// ok: package-import-check
	"fmt"
	// ruleid: package-import-check
	cs "github.com/mitchellh/copystructure"
	// ok: package-import-check
	"os"
	// ruleid: package-import-check
	log "github.com/prebid/prebid-server/v3/di"
)

import (
	// ok: package-import-check
	"fmt"
	// ruleid: package-import-check
	cs "github.com/mitchellh/copystructure/subpackage"
	// ok: package-import-check
	"os"
	// ruleid: package-import-check
	log "github.com/golang/glog/subpackage"
)

// ruleid: package-import-check
import "github.com/prebid/prebid-server/v3/di"

// ruleid: package-import-check
import "github.com/mitchellh/copystructure"

// ruleid: package-import-check
import log "github.com/prebid/prebid-server/v3/di"

// ruleid: package-import-check
import copy "github.com/mitchellh/copystructure"

// ok: package-import-check
import "fmt"  
