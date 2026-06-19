use fabric_sdk::prelude::*;

fn main() {
    fabric_sdk::chaincode::initialize()
        .register(
            "",
            functions![
                asset::create_asset,
                asset::asset_exists,
                asset::read_asset,
                asset::update_asset,
                asset::delete_asset,
                asset::transfer_asset,
                asset::get_all_assets
            ],
        )
        .launch();
}

pub mod asset {

    use fabric_sdk::prelude::*;
    use serde::{Deserialize, Serialize};

    #[derive(Debug, Serialize, Deserialize)]
    pub struct Asset {
        pub asset_id: String,
        pub color: String,
        pub size: i32,
        pub owner: String,
        pub appraised_value: i32,
    }

    /**
     * Creates a new asset on the ledger.
     *
     * `ctx the` transaction context
     * `asset_id `the ID of the new asset
     * `color` the color of the new asset
     * `size` the size for the new asset
     * `owner the owner` of the new asset
     * `appraised_value` the appraisedValue of the new asset
     * `return Asset` the created asset
     */
    #[transaction(CreateAsset)]
    pub async fn create_asset(
        ctx: Context,
        asset_id: String,
        color: String,
        size: i32,
        owner: String,
        appraised_value: i32,
    ) -> Asset {
        if asset_exists(ctx.clone(), asset_id.clone()).await {
            let error = format!("Asset {asset_id} already exists");
            println!("{}", error);
            panic!("{}", error);
        }
        put_asset(
            &ctx,
            Asset {
                asset_id,
                color,
                size,
                owner,
                appraised_value,
            },
        )
        .await
    }

    /**
     * Retrieves an asset with the specified ID from the ledger.
     *
     * `ctx` the transaction context
     * `assetID` the ID of the asset
     * `return Asset` the asset found on the ledger if there was one
     */
    #[transaction(ReadAsset)]
    pub async fn read_asset(ctx: Context, asset_id: String) -> Asset {
        serde_json::from_str(ctx.get_state_string(asset_id.as_str()).await.as_str())
            .expect("Invalid or no asset")
    }

    /**
     * Updates the properties of an asset on the ledger.
     *
     * `ctx` the transaction context
     * `assetID` the ID of the asset being updated
     * `color` the color of the asset being updated
     * `size` the size of the asset being updated
     * `owner` the owner of the asset being updated
     * `appraisedValue` the appraisedValue of the asset being updated
     * `return` the transferred asset
     */
    #[transaction(UpdateAsset)]
    pub async fn update_asset(
        ctx: Context,
        asset_id: String,
        color: String,
        size: i32,
        owner: String,
        appraised_value: i32,
    ) -> Asset {
        if !asset_exists(ctx.clone(), asset_id.clone()).await {
            let error = format!("Asset {asset_id} does not exists");
            println!("{}", error);
            panic!("{}", error);
        }
        put_asset(
            &ctx,
            Asset {
                asset_id,
                color,
                size,
                owner,
                appraised_value,
            },
        )
        .await
    }

    /**
     * Deletes asset on the ledger.
     *
     * `ctx` the transaction context
     * `assetID` the ID of the asset being deleted
     */
    #[transaction(DeleteAsset)]
    pub async fn delete_asset(ctx: Context, asset_id: String) {
        if !asset_exists(ctx.clone(), asset_id.clone()).await {
            let error = format!("Asset {asset_id} does not exists");
            println!("{}", error);
            panic!("{}", error);
        }
        ctx.del_state(asset_id.as_str()).await;
    }

    /**
     * Checks the existence of the asset on the ledger
     *
     * `ctx` the transaction context
     * `assetID` the ID of the asset
     * `return boolean` indicating the existence of the asset
     */
    #[transaction(AssetExists)]
    pub async fn asset_exists(ctx: Context, asset_id: String) -> bool {
        !ctx.get_state(&asset_id).await.is_empty()
    }

    /**
     * Changes the owner of a asset on the ledger.
     *
     * `ctx` the transaction context
     * `assetID` the ID of the asset being transferred
     * `newOwner` the new owner
     * `return` the old owner
     */
    #[transaction(TransferAsset)]
    pub async fn transfer_asset(ctx: Context, asset_id: String, new_owner: String) -> String {
        let asset = ctx.get_state_string(&asset_id).await;
        if asset.is_empty() {
            let error = format!("Asset {asset_id} does not exists");
            println!("{}", error);
            panic!("{}", error);
        }
        let asset: Asset = serde_json::from_str(&asset).expect("Invalid Asset");
        put_asset(
            &ctx,
            Asset {
                asset_id,
                color: asset.color,
                size: asset.size,
                owner: new_owner.clone(),
                appraised_value: asset.appraised_value,
            },
        )
        .await;
        new_owner
    }

    /**
     * Retrieves all assets from the ledger.
     *
     * `ctx` the transaction context
     * `return` array of assets found on the ledger
     */
    #[transaction(GetAllAssets)]
    pub async fn get_all_assets(ctx: Context) -> Vec<Asset> {
        ctx.get_state_by_range("", "")
            .await
            .map(|bytes| serde_json::from_slice(&bytes).expect("Invalid asset"))
            .collect()
    }

    async fn put_asset(ctx: &Context, asset: Asset) -> Asset {
        let storted_json = serde_json::to_string(&asset).expect("Couldn't serialize asset");
        ctx.put_state_string(&asset.asset_id, storted_json.as_str())
            .await;
        asset
    }
}
