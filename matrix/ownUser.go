package matrix

import (
	"strings"

	"github.com/therecipe/qt/gui"
	"github.com/tidwall/buntdb"
)

// getUserDisplayName returns the Dispaly name to a MXID
func getUserDisplayName(mxid string, cli *Client) (displayName string, err error) {
	// Get cache
	db.View(func(tx *Tx) error {
		tx.AscendKeys("user:displayName",
			func(key, value string) bool {
				displayName = value
				return true
			})
	})

	// If cache is empty query the api
	if displayName == "" {
		urlPath := cli.BuildURL("profile", mxid, "displayname")
		var resp *RespUserDisplayName
		_, err = cli.MakeRequest("GET", urlPath, nil, &resp)

		// Update cache
		err = db.Update(func(tx *buntdb.Tx) error {
			tx.Set("user:displayName", resp.DisplayName, nil)
			return nil
		})

		displayName = resp.DisplayName
	}
	return
}

// getOwnUserAvatar returns a *gui.QPixmap of an UserAvatar
func getOwnUserAvatar(cli *Client) *gui.QPixmap {
	// Init local vars
	var avatarData string
	var IMGdata string

	// Get cache
	db.View(func(tx *Tx) error {
		tx.AscendKeys("user:avatarData100x100",
			func(key, value string) bool {
				avatarData = value
				return true
			})
	})

	//If cache is empty do a ServerQuery
	if avatarData == "" {

		// Get avatarURL
		avatarURL, avatarErr := cli.GetAvatarURL()
		if avatarErr != nil {
			localLog.Println(avatarErr)
		}

		// If avatarURL is not empty (aka. has a avatar set) download it at the size of 100x100. Else make the data string empty
		if avatarURL != "" {
			hsURL := cli.HomeserverURL.String()
			avatarURL_splits := strings.Split(strings.Replace(avatarURL, "mxc://", "", -1), "/")

			urlPath := hsURL + "/_matrix/media/r0/thumbnail/" + avatarURL_splits[0] + "/" + avatarURL_splits[1] + "?width=100&height=100"

			data, err := cli.MakeRequest("GET", urlPath, nil, nil)
			if err != nil {
				localLog.Println(err)
			}
			IMGdata = string(data[:])
		} else {
			//TODO Generate default image (Step: AfterUI)
			IMGdata = ""
		}

		// Update cache
		DBerr := db.Update(func(tx *buntdb.Tx) error {
			tx.Set("user:avatarData100x100", IMGdata, nil)
			return nil
		})
		if DBerr != nil {
			localLog.Fatalln(err)
		}
	}

	// Convert avatarimage to QPixmap for usage in QT
	avatar := gui.NewQPixmap()
	avatar.LoadFromData(IMGdata, uint(len(IMGdata)), "", 0)
	return avatar
}